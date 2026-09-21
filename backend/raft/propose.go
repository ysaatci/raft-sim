package raft

import "fmt"

// NotLeaderError is returned when a proposal reaches a node that is not the
// leader. Leader is the node's best guess at the current leader, or None.
type NotLeaderError struct {
	Leader NodeID
}

func (e *NotLeaderError) Error() string {
	if e.Leader == None {
		return "raft: not leader, leader unknown"
	}
	return fmt.Sprintf("raft: not leader, try node %d", e.Leader)
}

// Propose appends a command to the log. Only the leader accepts proposals;
// the entry is committed once a majority has stored it.
func (n *Node) Propose(data string) error {
	if n.role != Leader {
		return &NotLeaderError{Leader: n.leader}
	}
	n.appendEntry(data)
	n.broadcastAppend()
	return nil
}

// appendEntry appends a new entry in the current term to the leader's log.
func (n *Node) appendEntry(data string) {
	e := Entry{Term: n.term, Index: n.log.lastIndex() + 1, Data: data}
	n.log.append(e)
	must(n.storage.Append([]Entry{e}))
	n.match[n.id] = e.Index
}
