package raft

import "testing"

// tickUntil ticks n until cond holds, failing after max ticks. It returns
// the number of ticks taken.
func tickUntil(t *testing.T, n *Node, max int, cond func() bool) int {
	t.Helper()
	for i := 1; i <= max; i++ {
		n.Tick()
		if cond() {
			return i
		}
	}
	t.Fatalf("condition not met after %d ticks", max)
	return 0
}

func TestElectionTimeoutIsRandomizedWithinRange(t *testing.T) {
	seen := map[int]bool{}
	for id := NodeID(1); id <= 20; id++ {
		n := newTestNode(t, 1, 3)
		n.rand = newTestNode(t, id, 20).rand // vary the seed
		n.resetTimers()
		to := n.Status().ElectionTimeout
		if to < 10 || to >= 20 {
			t.Fatalf("timeout %d outside [10, 20)", to)
		}
		seen[to] = true
	}
	if len(seen) < 3 {
		t.Fatalf("timeouts not randomized: %v", seen)
	}
}

func TestElectionTimeoutIsDeterministicForSeed(t *testing.T) {
	a, b := newTestNode(t, 2, 3), newTestNode(t, 2, 3)
	if a.Status().ElectionTimeout != b.Status().ElectionTimeout {
		t.Fatal("same seed produced different timeouts")
	}
}

func TestFollowerCampaignsAfterTimeout(t *testing.T) {
	n := newTestNode(t, 1, 3)
	timeout := n.Status().ElectionTimeout
	ticks := tickUntil(t, n, 20, func() bool { return n.Status().Role == Candidate })
	if ticks != timeout {
		t.Fatalf("campaigned after %d ticks, want %d", ticks, timeout)
	}
	st := n.Status()
	if st.Term != 1 || st.VotedFor != 1 {
		t.Fatalf("candidate state: %+v", st)
	}

	msgs := n.Ready().Messages
	if len(msgs) != 2 {
		t.Fatalf("sent %d messages, want 2", len(msgs))
	}
	for i, m := range msgs {
		if m.Type != MsgVote || m.To != NodeID(i+2) || m.Term != 1 {
			t.Errorf("message %d = %+v", i, m)
		}
	}
}

func TestCandidateRestartsElectionOnTimeout(t *testing.T) {
	n := newTestNode(t, 1, 3)
	tickUntil(t, n, 20, func() bool { return n.Status().Role == Candidate })
	tickUntil(t, n, 20, func() bool { return n.Status().Term == 2 })
	if n.Status().Role != Candidate {
		t.Fatalf("role = %v, want candidate", n.Status().Role)
	}
}

func TestCampaignPersistsVote(t *testing.T) {
	s := NewMemoryStorage()
	n := newTestNodeWithStorage(t, 1, 3, s)
	tickUntil(t, n, 20, func() bool { return n.Status().Role == Candidate })
	hs, _, _ := s.InitialState()
	if hs.Term != 1 || hs.VotedFor != 1 {
		t.Fatalf("persisted %+v, want term 1 votedFor 1", hs)
	}
}
