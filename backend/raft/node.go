package raft

import "math/rand/v2"

// Node is a single Raft participant. It is not safe for concurrent use;
// the caller drives it from one goroutine via Tick, Step and Ready.
type Node struct {
	id            NodeID
	peers         []NodeID
	electionTick  int
	heartbeatTick int
	rand          *rand.Rand
	storage       Storage

	role     Role
	term     uint64
	votedFor NodeID
	leader   NodeID
	log      *raftLog
	commit   uint64
	applied  uint64

	electionElapsed  int
	electionTimeout  int
	heartbeatElapsed int

	votes map[NodeID]bool // candidate only: who has granted us a vote

	msgs []Message
}

// Ready holds the output a node has produced since the last call to Ready.
type Ready struct {
	// Messages must be delivered to their recipients.
	Messages []Message
	// CommittedEntries must be applied to the state machine, in order.
	CommittedEntries []Entry
}

// Status is a read-only snapshot of a node's state.
type Status struct {
	ID              NodeID `json:"id"`
	Role            Role   `json:"role"`
	Term            uint64 `json:"term"`
	VotedFor        NodeID `json:"votedFor"`
	Leader          NodeID `json:"leader"`
	Commit          uint64 `json:"commit"`
	Applied         uint64 `json:"applied"`
	LastIndex       uint64 `json:"lastIndex"`
	LastTerm        uint64 `json:"lastTerm"`
	ElectionElapsed int    `json:"electionElapsed"`
	ElectionTimeout int    `json:"electionTimeout"`
}

// NewNode creates a node, restoring any state found in cfg.Storage.
func NewNode(cfg Config) (*Node, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	hs, ents, err := cfg.Storage.InitialState()
	if err != nil {
		return nil, err
	}
	n := &Node{
		id:            cfg.ID,
		peers:         append([]NodeID(nil), cfg.Peers...),
		electionTick:  cfg.ElectionTick,
		heartbeatTick: cfg.HeartbeatTick,
		rand:          cfg.Rand,
		storage:       cfg.Storage,
		log:           newLog(),
	}
	n.log.append(ents...)
	n.term, n.votedFor, n.commit = hs.Term, hs.VotedFor, hs.Commit
	n.resetTimers()
	return n, nil
}

// Tick advances the node's logical clock by one tick.
func (n *Node) Tick() {
	if n.role == Leader {
		return
	}
	n.electionElapsed++
	if n.electionElapsed >= n.electionTimeout {
		n.campaign()
	}
}

// Step processes a message from another node.
func (n *Node) Step(m Message) {
	switch {
	case m.Term > n.term:
		n.becomeFollower(m.Term, None)
	case m.Term < n.term:
		// Stale sender: reply so it learns the newer term and steps down.
		if m.Type == MsgVote {
			n.send(Message{Type: MsgVoteResp, To: m.From, Reject: true})
		}
		return
	}

	switch m.Type {
	case MsgVote:
		n.handleVote(m)
	}
}

// Ready returns and clears the node's pending output.
func (n *Node) Ready() Ready {
	rd := Ready{Messages: n.msgs}
	n.msgs = nil
	return rd
}

// Status returns a snapshot of the node's state.
func (n *Node) Status() Status {
	return Status{
		ID:              n.id,
		Role:            n.role,
		Term:            n.term,
		VotedFor:        n.votedFor,
		Leader:          n.leader,
		Commit:          n.commit,
		Applied:         n.applied,
		LastIndex:       n.log.lastIndex(),
		LastTerm:        n.log.lastTerm(),
		ElectionElapsed: n.electionElapsed,
		ElectionTimeout: n.electionTimeout,
	}
}

func (n *Node) becomeFollower(term uint64, leader NodeID) {
	if term != n.term {
		n.term = term
		n.votedFor = None
		n.persist()
	}
	n.role = Follower
	n.leader = leader
	n.resetTimers()
}

func (n *Node) resetTimers() {
	n.electionElapsed = 0
	n.heartbeatElapsed = 0
	// Randomized timeouts make split votes unlikely (Raft §5.2).
	n.electionTimeout = n.electionTick + n.rand.IntN(n.electionTick)
}

func (n *Node) send(m Message) {
	m.From = n.id
	m.Term = n.term
	n.msgs = append(n.msgs, m)
}

func (n *Node) persist() {
	must(n.storage.SetHardState(HardState{Term: n.term, VotedFor: n.votedFor, Commit: n.commit}))
}

// must panics on storage errors: a node that cannot persist its state
// cannot safely continue.
func must(err error) {
	if err != nil {
		panic("raft: storage failure: " + err.Error())
	}
}
