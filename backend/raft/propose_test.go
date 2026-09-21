package raft

import (
	"errors"
	"testing"
)

func TestLeaderAppendsNoopOnElection(t *testing.T) {
	n := leaderNode(t)
	if st := n.Status(); st.LastIndex != 1 || st.LastTerm != 1 {
		t.Fatalf("status = %+v, want no-op entry at (1, 1)", st)
	}
}

func TestProposeOnLeaderAppendsAndPersists(t *testing.T) {
	s := NewMemoryStorage()
	n := newTestNodeWithStorage(t, 1, 1, s) // single node: leader after one timeout
	tickUntil(t, n, 20, func() bool { return n.Status().Role == Leader })

	if err := n.Propose("set x 1"); err != nil {
		t.Fatal(err)
	}
	_, ents, _ := s.InitialState()
	want := Entry{Term: 1, Index: 2, Data: "set x 1"}
	if len(ents) != 2 || ents[1] != want {
		t.Fatalf("persisted entries = %+v, want no-op then %+v", ents, want)
	}
}

func TestProposeOnFollowerReturnsLeaderHint(t *testing.T) {
	n := newTestNode(t, 1, 3)
	var nle *NotLeaderError

	if err := n.Propose("x"); !errors.As(err, &nle) || nle.Leader != None {
		t.Fatalf("err = %v, want NotLeaderError with unknown leader", err)
	}
	n.Step(Message{Type: MsgApp, From: 3, To: 1, Term: 1})
	if err := n.Propose("x"); !errors.As(err, &nle) || nle.Leader != 3 {
		t.Fatalf("err = %v, want NotLeaderError pointing at 3", err)
	}
	if n.Status().LastIndex != 0 {
		t.Fatal("follower appended a proposal")
	}
}
