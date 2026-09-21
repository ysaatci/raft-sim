package raft

import (
	"errors"
	"fmt"
	"math/rand/v2"
)

// NodeID identifies a node in the cluster. IDs start at 1; None (0) means
// "no node", e.g. no vote cast or no known leader.
type NodeID int

// None is the zero NodeID.
const None NodeID = 0

// Role is the current Raft role of a node.
type Role int

const (
	Follower Role = iota
	Candidate
	Leader
)

func (r Role) String() string {
	switch r {
	case Follower:
		return "follower"
	case Candidate:
		return "candidate"
	case Leader:
		return "leader"
	}
	return fmt.Sprintf("Role(%d)", int(r))
}

// Entry is a single log entry. Index 0 is never a real entry.
type Entry struct {
	Term  uint64 `json:"term"`
	Index uint64 `json:"index"`
	Data  string `json:"data"`
}

// MsgType is the kind of a Raft message.
type MsgType int

const (
	MsgVote     MsgType = iota // RequestVote
	MsgVoteResp                // RequestVote response
	MsgApp                     // AppendEntries (also used as heartbeat)
	MsgAppResp                 // AppendEntries response
)

func (t MsgType) String() string {
	switch t {
	case MsgVote:
		return "RequestVote"
	case MsgVoteResp:
		return "RequestVoteResp"
	case MsgApp:
		return "AppendEntries"
	case MsgAppResp:
		return "AppendEntriesResp"
	}
	return fmt.Sprintf("MsgType(%d)", int(t))
}

// Message is exchanged between nodes. Only the fields relevant to Type are set.
type Message struct {
	Type MsgType `json:"type"`
	From NodeID  `json:"from"`
	To   NodeID  `json:"to"`
	Term uint64  `json:"term"`

	// MsgVote: the candidate's last log position.
	LastLogIndex uint64 `json:"lastLogIndex,omitempty"`
	LastLogTerm  uint64 `json:"lastLogTerm,omitempty"`

	// MsgApp: the entry preceding Entries, the entries, and the leader's commit.
	// MsgAppResp (reject): PrevLogIndex echoes the rejected request.
	PrevLogIndex uint64  `json:"prevLogIndex,omitempty"`
	PrevLogTerm  uint64  `json:"prevLogTerm,omitempty"`
	Entries      []Entry `json:"entries,omitempty"`
	Commit       uint64  `json:"commit,omitempty"`

	// Responses.
	Reject bool `json:"reject,omitempty"`
	// MsgAppResp (success): highest index known to match the leader.
	MatchIndex uint64 `json:"matchIndex,omitempty"`
	// MsgAppResp (reject): hints that let the leader skip back a whole term.
	ConflictIndex uint64 `json:"conflictIndex,omitempty"`
	ConflictTerm  uint64 `json:"conflictTerm,omitempty"`
}

// Config configures a Node.
type Config struct {
	// ID of this node. Must be listed in Peers.
	ID NodeID
	// Peers lists every node in the cluster, including this one.
	Peers []NodeID
	// ElectionTick is the minimum election timeout in ticks. The actual
	// timeout is drawn uniformly from [ElectionTick, 2*ElectionTick).
	ElectionTick int
	// HeartbeatTick is how often, in ticks, a leader sends heartbeats.
	// Must be smaller than ElectionTick.
	HeartbeatTick int
	// Rand supplies randomness. Seed it to make runs reproducible.
	Rand *rand.Rand
}

func (c *Config) validate() error {
	if c.ID == None {
		return errors.New("raft: ID must not be None")
	}
	found := false
	for _, p := range c.Peers {
		if p == c.ID {
			found = true
		}
	}
	if !found {
		return errors.New("raft: Peers must include ID")
	}
	if c.HeartbeatTick <= 0 {
		return errors.New("raft: HeartbeatTick must be positive")
	}
	if c.ElectionTick <= c.HeartbeatTick {
		return errors.New("raft: ElectionTick must be greater than HeartbeatTick")
	}
	if c.Rand == nil {
		return errors.New("raft: Rand must be set")
	}
	return nil
}
