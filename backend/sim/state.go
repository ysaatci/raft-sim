package sim

import (
	"cmp"
	"slices"

	"github.com/ysaatci/raft-sim/backend/raft"
)

// State is a complete, JSON-serializable snapshot of a cluster: everything
// the UI needs to draw one frame.
type State struct {
	Time       Time        `json:"time"`
	Config     Config      `json:"config"`
	Network    LinkConfig  `json:"network"` // current conditions of links without an override
	Nodes      []NodeView  `json:"nodes"`
	Flights    []Flight    `json:"flights"` // in flight or recently dropped, by ID
	Links      []LinkView  `json:"links"`   // every directed link between two nodes
	Violations []Violation `json:"violations"`
}

// NodeView describes one node. For a crashed node, Status holds only what
// survived on disk (term, vote, commit and log).
type NodeView struct {
	raft.Status
	State NodeState              `json:"state"`
	Log   []raft.Entry           `json:"log"`
	Next  map[raft.NodeID]uint64 `json:"next,omitempty"`  // leader only
	Match map[raft.NodeID]uint64 `json:"match,omitempty"` // leader only
	KV    map[string]string      `json:"kv"`
	Inbox int                    `json:"inbox"` // messages queued while paused
}

// LinkView is the configuration of the link From -> To.
type LinkView struct {
	From raft.NodeID `json:"from"`
	To   raft.NodeID `json:"to"`
	LinkConfig
}

// State returns a snapshot of the whole cluster.
func (c *Cluster) State() State {
	s := State{
		Time:       c.now,
		Config:     c.cfg,
		Network:    c.net.Defaults(),
		Violations: append([]Violation{}, c.check.violations...),
		Flights:    []Flight{},
	}
	for _, sn := range c.nodes {
		s.Nodes = append(s.Nodes, c.nodeView(sn))
		for _, to := range c.nodes {
			if sn != to {
				s.Links = append(s.Links, LinkView{From: sn.id, To: to.id, LinkConfig: c.net.Link(sn.id, to.id)})
			}
		}
	}
	c.inflight.each(func(_ Time, f *Flight) { s.Flights = append(s.Flights, *f) })
	for _, f := range c.dropped {
		s.Flights = append(s.Flights, *f)
	}
	slices.SortFunc(s.Flights, func(a, b Flight) int { return cmp.Compare(a.ID, b.ID) })
	return s
}

func (c *Cluster) nodeView(sn *simNode) NodeView {
	v := NodeView{State: sn.state, KV: sn.kv.Data(), Inbox: len(sn.inbox)}
	if sn.node != nil {
		v.Status = sn.node.Status()
		v.Log = sn.node.Entries()
		v.Next, v.Match = sn.node.Progress()
	} else {
		hs, ents, _ := sn.storage.InitialState()
		v.Status = raft.Status{ID: sn.id, Term: hs.Term, VotedFor: hs.VotedFor, Commit: hs.Commit}
		v.Log = ents
		if n := len(ents); n > 0 {
			v.LastIndex, v.LastTerm = ents[n-1].Index, ents[n-1].Term
		}
	}
	if v.Log == nil {
		v.Log = []raft.Entry{} // serialize as [], not null
	}
	return v
}
