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

// app builds an AppendEntries from leader 2 in term 3.
func app(prevIndex, prevTerm, commit uint64, ents ...Entry) Message {
	return Message{Type: MsgApp, From: 2, To: 1, Term: 3, PrevLogIndex: prevIndex, PrevLogTerm: prevTerm, Entries: ents, Commit: commit}
}

func ent(index, term uint64) Entry { return Entry{Term: term, Index: index} }

// followerWithLog returns a follower whose log (memory and storage) has
// entry i+1 in terms[i].
func followerWithLog(t *testing.T, terms ...uint64) (*Node, *MemoryStorage) {
	t.Helper()
	s := NewMemoryStorage()
	var ents []Entry
	for i, tm := range terms {
		ents = append(ents, ent(uint64(i+1), tm))
	}
	s.Append(ents)
	return newTestNodeWithStorage(t, 1, 3, s), s
}

func logTerms(n *Node) []uint64 {
	var out []uint64
	for _, e := range n.log.entries[1:] {
		out = append(out, e.Term)
	}
	return out
}

func equalTerms(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestAppendRejectsWhenPrevMissing(t *testing.T) {
	n, _ := followerWithLog(t, 1, 1)
	n.Step(app(5, 2, 0))
	resp := onlyMsg(t, n)
	if !resp.Reject || resp.ConflictIndex != 3 || resp.ConflictTerm != 0 || resp.PrevLogIndex != 5 {
		t.Fatalf("resp = %+v, want reject with conflictIndex 3", resp)
	}
}

func TestAppendRejectsOnPrevTermMismatchWithHint(t *testing.T) {
	n, _ := followerWithLog(t, 1, 2, 2, 2)
	n.Step(app(4, 3, 0))
	resp := onlyMsg(t, n)
	if !resp.Reject || resp.ConflictTerm != 2 || resp.ConflictIndex != 2 {
		t.Fatalf("resp = %+v, want reject with conflict term 2 starting at 2", resp)
	}
}

func TestAppendAddsEntries(t *testing.T) {
	n, s := followerWithLog(t, 1)
	n.Step(app(1, 1, 0, ent(2, 3), ent(3, 3)))
	resp := onlyMsg(t, n)
	if resp.Reject || resp.MatchIndex != 3 {
		t.Fatalf("resp = %+v, want success with matchIndex 3", resp)
	}
	if got := logTerms(n); !equalTerms(got, []uint64{1, 3, 3}) {
		t.Fatalf("log terms = %v", got)
	}
	if _, ents, _ := s.InitialState(); len(ents) != 3 {
		t.Fatalf("storage has %d entries, want 3", len(ents))
	}
}

func TestAppendTruncatesConflictingSuffix(t *testing.T) {
	n, s := followerWithLog(t, 1, 2, 2)
	n.Step(app(1, 1, 0, ent(2, 3)))
	if got := logTerms(n); !equalTerms(got, []uint64{1, 3}) {
		t.Fatalf("log terms = %v, want [1 3]", got)
	}
	if _, ents, _ := s.InitialState(); len(ents) != 2 || ents[1].Term != 3 {
		t.Fatalf("storage = %+v", ents)
	}
}

func TestDelayedAppendDoesNotTruncateMatchingEntries(t *testing.T) {
	n, _ := followerWithLog(t, 1)
	n.Step(app(1, 1, 0, ent(2, 3), ent(3, 3)))
	n.Step(app(1, 1, 0, ent(2, 3))) // older, shorter message arrives late
	if got := logTerms(n); !equalTerms(got, []uint64{1, 3, 3}) {
		t.Fatalf("log terms = %v, delayed message truncated the log", got)
	}
}

func TestAppendAdvancesCommitUpToLastNewEntry(t *testing.T) {
	n, s := followerWithLog(t, 1, 1, 1)
	// Leader has committed 5, but this message only vouches for index 2.
	n.Step(app(1, 1, 5, ent(2, 1)))
	if n.Status().Commit != 2 {
		t.Fatalf("commit = %d, want 2", n.Status().Commit)
	}
	if hs, _, _ := s.InitialState(); hs.Commit != 2 {
		t.Fatalf("persisted commit = %d, want 2", hs.Commit)
	}
	// A delayed heartbeat must never move commit backwards.
	n.Step(app(1, 1, 1))
	if n.Status().Commit != 2 {
		t.Fatalf("commit moved backwards to %d", n.Status().Commit)
	}
}
