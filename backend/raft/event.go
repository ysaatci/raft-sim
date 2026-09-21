package raft

// EventType names a notable state change, for display in the UI.
type EventType string

const (
	EventElectionStarted EventType = "election-started" // became candidate
	EventBecameLeader    EventType = "became-leader"
	EventSteppedDown     EventType = "stepped-down"    // leader or candidate became follower
	EventVoteGranted     EventType = "vote-granted"    // Peer received our vote
	EventLogTruncated    EventType = "log-truncated"   // entries from Index on were discarded
	EventCommitAdvanced  EventType = "commit-advanced" // commit index moved to Index
)

// Event describes a state change on Node during Term.
type Event struct {
	Type  EventType `json:"type"`
	Node  NodeID    `json:"node"`
	Term  uint64    `json:"term"`
	Peer  NodeID    `json:"peer,omitempty"`
	Index uint64    `json:"index,omitempty"`
}

func (n *Node) emit(e Event) {
	e.Node = n.id
	e.Term = n.term
	n.events = append(n.events, e)
}
