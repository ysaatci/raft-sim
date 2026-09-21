package sim

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/ysaatci/raft-sim/backend/raft"
)

// Config describes a simulated cluster.
type Config struct {
	Size          int        `json:"size"`
	Seed          uint64     `json:"seed"`
	TickMs        int        `json:"tickMs"`        // virtual ms per Raft tick
	ElectionTick  int        `json:"electionTick"`  // in ticks
	HeartbeatTick int        `json:"heartbeatTick"` // in ticks
	Network       LinkConfig `json:"network"`       // default for every link
}

// DefaultConfig is a 5-node cluster with timings close to the Raft paper:
// 50ms heartbeats, 150-300ms election timeouts, ~20ms one-way latency.
func DefaultConfig() Config {
	return Config{
		Size:          5,
		Seed:          1,
		TickMs:        10,
		ElectionTick:  15,
		HeartbeatTick: 5,
		Network:       LinkConfig{LatencyMs: 20, JitterMs: 5},
	}
}

// NodeState is whether a simulated node is running.
type NodeState string

const (
	NodeUp      NodeState = "up"
	NodeCrashed NodeState = "crashed" // process gone; storage survives
	NodePaused  NodeState = "paused"  // frozen, e.g. a long GC pause; messages queue up
)

// Simulator events, reported alongside the Raft events from each node.
const (
	EventNodeCrashed   raft.EventType = "node-crashed"
	EventNodeRestarted raft.EventType = "node-restarted"
	EventNodePaused    raft.EventType = "node-paused"
	EventNodeResumed   raft.EventType = "node-resumed"
)

// TimedEvent is an event stamped with the virtual time it happened at.
type TimedEvent struct {
	Time Time `json:"time"`
	raft.Event
}

// Flight is a message travelling through the network.
type Flight struct {
	ID        uint64       `json:"id"`
	Msg       raft.Message `json:"msg"`
	SentAt    Time         `json:"sentAt"`
	DeliverAt Time         `json:"deliverAt"` // for a dropped message: when it would have arrived
	Dropped   bool         `json:"dropped,omitempty"`
}

type simNode struct {
	id          raft.NodeID
	state       NodeState
	node        *raft.Node // nil while crashed
	storage     *raft.MemoryStorage
	tickOffset  Time // nodes' clocks are not aligned
	incarnation uint64
	inbox       []raft.Message // delivered while paused
	kv          *KV            // volatile: rebuilt from the log on restart
}

// Cluster is a Raft cluster running on a simulated network in virtual time.
// It is not safe for concurrent use.
type Cluster struct {
	cfg      Config
	now      Time
	net      *Network
	nodes    []*simNode // nodes[i] has ID i+1
	inflight queue[*Flight]
	dropped  []*Flight // lost messages, kept until they would have arrived
	nextID   uint64
	events   []TimedEvent
	check    *checker
}

// NewCluster builds a cluster with every node up and no leader yet.
func NewCluster(cfg Config) (*Cluster, error) {
	if cfg.Size < 1 || cfg.Size > 9 {
		return nil, errors.New("sim: Size must be between 1 and 9")
	}
	if cfg.TickMs < 1 {
		return nil, errors.New("sim: TickMs must be positive")
	}
	c := &Cluster{
		cfg:   cfg,
		net:   NewNetwork(rand.New(rand.NewPCG(cfg.Seed, 0)), cfg.Network),
		check: newChecker(),
	}
	offsets := rand.New(rand.NewPCG(cfg.Seed, 1))
	for i := range cfg.Size {
		sn := &simNode{
			id:         raft.NodeID(i + 1),
			storage:    raft.NewMemoryStorage(),
			tickOffset: Time(offsets.IntN(cfg.TickMs)),
		}
		if err := c.boot(sn); err != nil {
			return nil, err
		}
		c.nodes = append(c.nodes, sn)
	}
	return c, nil
}

// boot starts (or restarts) a node from its storage.
func (c *Cluster) boot(sn *simNode) error {
	peers := make([]raft.NodeID, c.cfg.Size)
	for i := range peers {
		peers[i] = raft.NodeID(i + 1)
	}
	// Each incarnation gets its own deterministic random stream.
	seed2 := uint64(sn.id)<<32 | sn.incarnation
	n, err := raft.NewNode(raft.Config{
		ID:            sn.id,
		Peers:         peers,
		ElectionTick:  c.cfg.ElectionTick,
		HeartbeatTick: c.cfg.HeartbeatTick,
		Rand:          rand.New(rand.NewPCG(c.cfg.Seed, seed2)),
		Storage:       sn.storage,
	})
	if err != nil {
		return err
	}
	sn.node = n
	sn.state = NodeUp
	sn.kv = NewKV()
	return nil
}

// Now returns the current virtual time.
func (c *Cluster) Now() Time { return c.now }

// Advance runs the simulation forward by ms milliseconds.
func (c *Cluster) Advance(ms Time) {
	for range ms {
		c.now++
		c.deliverDue()
		c.tickDue()
	}
	c.dropped = slices.DeleteFunc(c.dropped, func(f *Flight) bool { return f.DeliverAt < c.now })
	c.checkLogs()
}

func (c *Cluster) deliverDue() {
	for {
		f, ok := c.inflight.popDue(c.now)
		if !ok {
			return
		}
		m := f.Msg
		if c.net.Link(m.From, m.To).Cut {
			continue // link was cut while the message was in flight
		}
		sn := c.node(m.To)
		switch sn.state {
		case NodeUp:
			sn.node.Step(m)
			c.collect(sn)
		case NodePaused:
			sn.inbox = append(sn.inbox, m)
		}
	}
}

func (c *Cluster) tickDue() {
	tick := Time(c.cfg.TickMs)
	for _, sn := range c.nodes {
		if sn.state == NodeUp && (c.now-sn.tickOffset)%tick == 0 {
			sn.node.Tick()
			c.collect(sn)
		}
	}
}

// collect routes a node's outbound messages and records its events.
func (c *Cluster) collect(sn *simNode) {
	rd := sn.node.Ready()
	for _, m := range rd.Messages {
		c.nextID++
		at, ok := c.net.route(m.From, m.To, c.now)
		f := &Flight{ID: c.nextID, Msg: m, SentAt: c.now, DeliverAt: at, Dropped: !ok}
		if ok {
			c.inflight.push(at, f)
		} else {
			c.dropped = append(c.dropped, f)
		}
	}
	for _, e := range rd.CommittedEntries {
		c.check.onApply(c.now, sn.id, e)
		sn.kv.Apply(e.Data)
	}
	for _, e := range rd.Events {
		c.events = append(c.events, TimedEvent{Time: c.now, Event: e})
		switch e.Type {
		case raft.EventBecameLeader:
			c.check.onLeader(c.now, sn.id, e.Term, sn.node.Entries())
		case raft.EventCommitAdvanced:
			c.check.onCommit(c.now, sn.id, e.Term, sn.node.Entries(), e.Index, c.leaderViews())
		}
	}
}

// running returns the nodes that have in-memory state (up or paused).
func (c *Cluster) running() []*simNode {
	var out []*simNode
	for _, sn := range c.nodes {
		if sn.node != nil {
			out = append(out, sn)
		}
	}
	return out
}

func (c *Cluster) leaderViews() []leaderView {
	var out []leaderView
	for _, sn := range c.running() {
		if st := sn.node.Status(); st.Role == raft.Leader {
			out = append(out, leaderView{id: sn.id, term: st.Term, entries: sn.node.Entries()})
		}
	}
	return out
}

func (c *Cluster) checkLogs() {
	logs := map[raft.NodeID][]raft.Entry{}
	var ids []raft.NodeID
	for _, sn := range c.running() {
		logs[sn.id] = sn.node.Entries()
		ids = append(ids, sn.id)
	}
	c.check.checkLogs(c.now, logs, ids)
}

// Violations returns every safety violation observed so far. In a correct
// Raft implementation it is always empty.
func (c *Cluster) Violations() []Violation { return c.check.violations }

func (c *Cluster) emit(id raft.NodeID, t raft.EventType) {
	sn := c.node(id)
	var term uint64
	if sn.node != nil {
		term = sn.node.Status().Term
	}
	c.events = append(c.events, TimedEvent{Time: c.now, Event: raft.Event{Type: t, Node: id, Term: term}})
}

func (c *Cluster) node(id raft.NodeID) *simNode { return c.nodes[id-1] }

func (c *Cluster) lookup(id raft.NodeID) (*simNode, error) {
	if id < 1 || int(id) > len(c.nodes) {
		return nil, fmt.Errorf("sim: no node %d", id)
	}
	return c.node(id), nil
}

// Crash stops a node. Its persisted state survives; everything else,
// including messages addressed to it, is lost.
func (c *Cluster) Crash(id raft.NodeID) error {
	sn, err := c.lookup(id)
	if err != nil || sn.state == NodeCrashed {
		return err
	}
	c.emit(id, EventNodeCrashed)
	sn.state = NodeCrashed
	sn.node = nil
	sn.inbox = nil
	return nil
}

// Restart boots a crashed node from its persisted state.
func (c *Cluster) Restart(id raft.NodeID) error {
	sn, err := c.lookup(id)
	if err != nil || sn.state != NodeCrashed {
		return err
	}
	sn.incarnation++
	if err := c.boot(sn); err != nil {
		return err
	}
	c.emit(id, EventNodeRestarted)
	return nil
}

// Pause freezes a node: it stops ticking and queues incoming messages.
func (c *Cluster) Pause(id raft.NodeID) error {
	sn, err := c.lookup(id)
	if err != nil || sn.state != NodeUp {
		return err
	}
	sn.state = NodePaused
	c.emit(id, EventNodePaused)
	return nil
}

// Resume unfreezes a paused node, which then processes its queued messages.
func (c *Cluster) Resume(id raft.NodeID) error {
	sn, err := c.lookup(id)
	if err != nil || sn.state != NodePaused {
		return err
	}
	sn.state = NodeUp
	c.emit(id, EventNodeResumed)
	for _, m := range sn.inbox {
		sn.node.Step(m)
	}
	sn.inbox = nil
	c.collect(sn)
	return nil
}

// Propose submits a client command to node id. It fails with a
// *raft.NotLeaderError if that node is not the leader.
func (c *Cluster) Propose(id raft.NodeID, data string) error {
	sn, err := c.lookup(id)
	if err != nil {
		return err
	}
	if sn.state != NodeUp {
		return fmt.Errorf("sim: node %d is %s", id, sn.state)
	}
	if err := sn.node.Propose(data); err != nil {
		return err
	}
	c.collect(sn)
	return nil
}

// SetLink overrides the link from -> to.
func (c *Cluster) SetLink(from, to raft.NodeID, cfg LinkConfig) { c.net.SetLink(from, to, cfg) }

// SetNetwork changes the default configuration of every link.
func (c *Cluster) SetNetwork(cfg LinkConfig) { c.net.SetDefaults(cfg) }

// Partition cuts every link between nodes in different groups.
func (c *Cluster) Partition(groups ...[]raft.NodeID) { c.net.Partition(groups...) }

// Heal reconnects every cut link.
func (c *Cluster) Heal() { c.net.Heal() }

// Leader returns the running leader with the highest term, or None. During
// a partition an old leader may still believe it leads; it is ignored.
func (c *Cluster) Leader() raft.NodeID {
	leader, term := raft.None, uint64(0)
	for _, sn := range c.nodes {
		if sn.state == NodeUp {
			if st := sn.node.Status(); st.Role == raft.Leader && st.Term >= term {
				leader, term = sn.id, st.Term
			}
		}
	}
	return leader
}

// NodeState reports whether node id is up, crashed or paused.
func (c *Cluster) NodeState(id raft.NodeID) NodeState { return c.node(id).state }

// Status returns node id's Raft status, or false if it is crashed.
func (c *Cluster) Status(id raft.NodeID) (raft.Status, bool) {
	sn := c.node(id)
	if sn.node == nil {
		return raft.Status{}, false
	}
	return sn.node.Status(), true
}

// KV returns node id's state machine. A crashed node keeps the state it
// had when it crashed until it restarts.
func (c *Cluster) KV(id raft.NodeID) *KV { return c.node(id).kv }

// Events returns and clears the events recorded since the last call.
func (c *Cluster) Events() []TimedEvent {
	evs := c.events
	c.events = nil
	return evs
}
