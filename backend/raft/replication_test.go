package raft

import "testing"

// leaderNode returns node 1 of a 3-node cluster as leader of term 1, with
// its pending messages drained.
func leaderNode(t *testing.T) *Node {
	t.Helper()
	n := campaignNode(t, 3)
	n.Step(voteResp(2, 1, false))
	if n.Status().Role != Leader {
		t.Fatal("setup: node did not become leader")
	}
	n.Ready()
	return n
}

func TestNewLeaderSendsHeartbeatsImmediately(t *testing.T) {
	n := campaignNode(t, 3)
	n.Step(voteResp(2, 1, false))
	msgs := n.Ready().Messages
	if len(msgs) != 2 {
		t.Fatalf("sent %d messages, want 2", len(msgs))
	}
	for _, m := range msgs {
		if m.Type != MsgApp || m.Term != 1 {
			t.Errorf("unexpected message %+v", m)
		}
	}
}

func TestLeaderSendsHeartbeatEveryHeartbeatTick(t *testing.T) {
	n := leaderNode(t) // HeartbeatTick = 1
	for range 3 {
		n.Tick()
		if got := len(n.Ready().Messages); got != 2 {
			t.Fatalf("heartbeat round sent %d messages, want 2", got)
		}
	}
	if n.Status().Role != Leader {
		t.Fatal("leader must not time out")
	}
}

func TestHeartbeatResetsFollowerElectionTimer(t *testing.T) {
	n := newTestNode(t, 1, 3)
	timeout := n.Status().ElectionTimeout
	for i := 0; i < 3*timeout; i++ {
		n.Tick()
		if i%5 == 0 {
			n.Step(Message{Type: MsgApp, From: 2, To: 1, Term: 1})
		}
	}
	st := n.Status()
	if st.Role != Follower || st.Leader != 2 {
		t.Fatalf("status = %+v, want follower of 2", st)
	}
	resp := n.Ready().Messages
	if len(resp) == 0 || resp[0].Type != MsgAppResp || resp[0].Reject {
		t.Fatalf("expected heartbeat acks, got %+v", resp)
	}
}

func TestCandidateStepsDownOnAppendFromSameTerm(t *testing.T) {
	n := campaignNode(t, 3)
	n.Step(Message{Type: MsgApp, From: 2, To: 1, Term: 1})
	if st := n.Status(); st.Role != Follower || st.Leader != 2 || st.Term != 1 {
		t.Fatalf("status = %+v, want follower of 2 in term 1", st)
	}
}

func TestStaleAppendIsRejectedWithCurrentTerm(t *testing.T) {
	n := newTestNode(t, 1, 3)
	n.Step(Message{Type: MsgApp, From: 2, To: 1, Term: 3})
	n.Ready()
	n.Step(Message{Type: MsgApp, From: 3, To: 1, Term: 2})
	resp := onlyMsg(t, n)
	if resp.Type != MsgAppResp || !resp.Reject || resp.Term != 3 || resp.To != 3 {
		t.Fatalf("resp = %+v, want reject in term 3", resp)
	}
	if n.Status().Leader != 2 {
		t.Fatal("stale append changed the known leader")
	}
}
