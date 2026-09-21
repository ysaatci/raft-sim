package raft

import (
	"math/rand/v2"
	"testing"
)

// newTestNode builds node id in a cluster of peers 1..size.
func newTestNode(t *testing.T, id NodeID, size int) *Node {
	t.Helper()
	return newTestNodeWithStorage(t, id, size, NewMemoryStorage())
}

func newTestNodeWithStorage(t *testing.T, id NodeID, size int, s Storage) *Node {
	t.Helper()
	peers := make([]NodeID, size)
	for i := range peers {
		peers[i] = NodeID(i + 1)
	}
	n, err := NewNode(Config{
		ID:            id,
		Peers:         peers,
		ElectionTick:  10,
		HeartbeatTick: 1,
		Rand:          rand.New(rand.NewPCG(uint64(id), 42)),
		Storage:       s,
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestNewNodeStartsAsFollower(t *testing.T) {
	n := newTestNode(t, 1, 3)
	st := n.Status()
	if st.Role != Follower || st.Term != 0 || st.VotedFor != None || st.Leader != None {
		t.Fatalf("unexpected initial status: %+v", st)
	}
}

func TestNewNodeRejectsInvalidConfig(t *testing.T) {
	if _, err := NewNode(Config{}); err == nil {
		t.Fatal("expected error for empty config")
	}
}

func TestStepHigherTermUpdatesAndPersistsTerm(t *testing.T) {
	s := NewMemoryStorage()
	n := newTestNodeWithStorage(t, 1, 3, s)
	n.Step(Message{Type: MsgApp, From: 2, To: 1, Term: 5})
	if n.Status().Term != 5 {
		t.Fatalf("term = %d, want 5", n.Status().Term)
	}
	if hs, _, _ := s.InitialState(); hs.Term != 5 {
		t.Fatalf("persisted term = %d, want 5", hs.Term)
	}
}

func TestReadyDrainsMessages(t *testing.T) {
	n := newTestNode(t, 1, 3)
	n.send(Message{Type: MsgVote, To: 2})
	if rd := n.Ready(); len(rd.Messages) != 1 || rd.Messages[0].From != 1 {
		t.Fatalf("first Ready = %+v", rd)
	}
	if rd := n.Ready(); len(rd.Messages) != 0 {
		t.Fatalf("second Ready should be empty, got %+v", rd)
	}
}

func TestReadyReturnsCommittedEntriesOnce(t *testing.T) {
	n := newTestNode(t, 1, 3)
	n.Step(Message{Type: MsgApp, From: 2, To: 1, Term: 1, Commit: 2,
		Entries: []Entry{{Term: 1, Index: 1, Data: "a"}, {Term: 1, Index: 2, Data: "b"}, {Term: 1, Index: 3, Data: "c"}}})

	rd := n.Ready()
	if len(rd.CommittedEntries) != 2 || rd.CommittedEntries[0].Data != "a" || rd.CommittedEntries[1].Data != "b" {
		t.Fatalf("committed = %+v, want entries a, b", rd.CommittedEntries)
	}
	if n.Status().Applied != 2 {
		t.Fatalf("applied = %d, want 2", n.Status().Applied)
	}
	if rd := n.Ready(); len(rd.CommittedEntries) != 0 {
		t.Fatalf("entries returned twice: %+v", rd.CommittedEntries)
	}

	n.Step(Message{Type: MsgApp, From: 2, To: 1, Term: 1, PrevLogIndex: 3, PrevLogTerm: 1, Commit: 3})
	if rd := n.Ready(); len(rd.CommittedEntries) != 1 || rd.CommittedEntries[0].Data != "c" {
		t.Fatalf("committed = %+v, want entry c", rd.CommittedEntries)
	}
}
