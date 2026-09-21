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

func voteReq(from NodeID, term, lastIndex, lastTerm uint64) Message {
	return Message{Type: MsgVote, From: from, To: 1, Term: term, LastLogIndex: lastIndex, LastLogTerm: lastTerm}
}

// onlyMsg returns the single message the node has sent, failing otherwise.
func onlyMsg(t *testing.T, n *Node) Message {
	t.Helper()
	msgs := n.Ready().Messages
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1: %+v", len(msgs), msgs)
	}
	return msgs[0]
}

func TestVoteGrantedToUpToDateCandidate(t *testing.T) {
	s := NewMemoryStorage()
	n := newTestNodeWithStorage(t, 1, 3, s)
	n.Step(voteReq(2, 1, 0, 0))
	resp := onlyMsg(t, n)
	if resp.Type != MsgVoteResp || resp.To != 2 || resp.Term != 1 || resp.Reject {
		t.Fatalf("resp = %+v, want granted vote", resp)
	}
	if hs, _, _ := s.InitialState(); hs.VotedFor != 2 {
		t.Fatalf("vote not persisted: %+v", hs)
	}
}

func TestOneVotePerTerm(t *testing.T) {
	n := newTestNode(t, 1, 3)
	n.Step(voteReq(2, 1, 0, 0))
	n.Ready()
	n.Step(voteReq(3, 1, 0, 0))
	if resp := onlyMsg(t, n); !resp.Reject {
		t.Fatal("second candidate in same term got a vote")
	}
	// A retransmitted request from the candidate we voted for is granted again.
	n.Step(voteReq(2, 1, 0, 0))
	if resp := onlyMsg(t, n); resp.Reject {
		t.Fatal("repeat request from same candidate rejected")
	}
}

func TestVoteRejectedForStaleTerm(t *testing.T) {
	n := newTestNode(t, 1, 3)
	n.Step(voteReq(2, 5, 0, 0))
	n.Ready()
	n.Step(voteReq(3, 4, 0, 0))
	resp := onlyMsg(t, n)
	if !resp.Reject || resp.Term != 5 {
		t.Fatalf("resp = %+v, want reject carrying term 5", resp)
	}
}

func TestVoteRejectedWhenCandidateLogBehind(t *testing.T) {
	cases := map[string]struct{ lastIndex, lastTerm uint64 }{
		"lower last term":        {5, 1},
		"same term, shorter log": {1, 2},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			n := newTestNode(t, 1, 3)
			n.log = logWithTerms(1, 2) // last = (2, 2)
			n.Step(voteReq(2, 3, c.lastIndex, c.lastTerm))
			if resp := onlyMsg(t, n); !resp.Reject {
				t.Fatal("vote granted to candidate with stale log")
			}
			if n.Status().Term != 3 {
				t.Fatal("higher term should still be adopted")
			}
		})
	}
}

func TestHigherTermVoteMakesCandidateStepDown(t *testing.T) {
	n := newTestNode(t, 1, 3)
	tickUntil(t, n, 20, func() bool { return n.Status().Role == Candidate })
	n.Ready()
	n.Step(voteReq(2, 2, 0, 0))
	st := n.Status()
	if st.Role != Follower || st.Term != 2 || st.VotedFor != 2 {
		t.Fatalf("status = %+v, want follower in term 2 voting for 2", st)
	}
}

func TestGrantingVoteResetsElectionTimer(t *testing.T) {
	n := newTestNode(t, 1, 3)
	for range 5 {
		n.Tick()
	}
	n.Step(voteReq(2, 1, 0, 0))
	if n.Status().ElectionElapsed != 0 {
		t.Fatal("election timer not reset after granting vote")
	}
}
