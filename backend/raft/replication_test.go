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

// testCluster is a minimal lossless network for driving a few nodes.
type testCluster struct {
	t        *testing.T
	nodes    map[NodeID]*Node
	storages map[NodeID]*MemoryStorage
}

// newTestCluster builds nodes 1..len(logs); node i starts with log terms
// logs[i-1] and a current term equal to its highest log term.
func newTestCluster(t *testing.T, logs ...[]uint64) *testCluster {
	t.Helper()
	c := &testCluster{t: t, nodes: map[NodeID]*Node{}, storages: map[NodeID]*MemoryStorage{}}
	for i, terms := range logs {
		s := NewMemoryStorage()
		var ents []Entry
		var term uint64
		for j, tm := range terms {
			ents = append(ents, ent(uint64(j+1), tm))
			term = max(term, tm)
		}
		s.Append(ents)
		s.SetHardState(HardState{Term: term})
		id := NodeID(i + 1)
		c.nodes[id] = newTestNodeWithStorage(t, id, len(logs), s)
		c.storages[id] = s
	}
	return c
}

// deliver passes messages between nodes until the network is quiet.
func (c *testCluster) deliver() {
	c.t.Helper()
	for round := 0; round < 100; round++ {
		var msgs []Message
		for id := NodeID(1); int(id) <= len(c.nodes); id++ {
			msgs = append(msgs, c.nodes[id].Ready().Messages...)
		}
		if len(msgs) == 0 {
			return
		}
		for _, m := range msgs {
			c.nodes[m.To].Step(m)
		}
	}
	c.t.Fatal("network did not go quiet")
}

// elect ticks node id alone until it campaigns, then delivers all traffic.
func (c *testCluster) elect(id NodeID) {
	c.t.Helper()
	n := c.nodes[id]
	tickUntil(c.t, n, 100, func() bool { return n.Status().Role == Candidate })
	c.deliver()
	if n.Status().Role != Leader {
		c.t.Fatalf("node %d failed to become leader", id)
	}
}

func TestLeaderFirstAppendCarriesNoop(t *testing.T) {
	n := campaignNode(t, 3)
	n.Step(voteResp(2, 1, false))
	for _, m := range n.Ready().Messages {
		if m.PrevLogIndex != 0 || len(m.Entries) != 1 || m.Entries[0].Index != 1 {
			t.Errorf("append = %+v, want the no-op after index 0", m)
		}
	}
}

func TestAppendSuccessAdvancesProgress(t *testing.T) {
	n := leaderNode(t)
	n.Step(Message{Type: MsgAppResp, From: 2, To: 1, Term: 1, MatchIndex: 1})
	if n.match[2] != 1 || n.next[2] != 2 {
		t.Fatalf("match=%d next=%d, want 1 and 2", n.match[2], n.next[2])
	}
	// A delayed, smaller ack must not move progress backwards.
	n.Step(Message{Type: MsgAppResp, From: 2, To: 1, Term: 1, MatchIndex: 0})
	if n.match[2] != 1 || n.next[2] != 2 {
		t.Fatalf("progress regressed: match=%d next=%d", n.match[2], n.next[2])
	}
}

func TestAppendRejectBacksUpWithHints(t *testing.T) {
	cases := map[string]struct {
		conflictIndex, conflictTerm, wantNext uint64
	}{
		"follower log too short":            {2, 0, 2},
		"conflict term unknown to leader":   {3, 3, 3},
		"leader has entries from that term": {2, 2, 4},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := newTestCluster(t, []uint64{1, 2, 2, 4}, nil, nil)
			n := c.nodes[1]
			tickUntil(t, n, 100, func() bool { return n.Status().Role == Candidate })
			n.Step(voteResp(2, n.Status().Term, false))
			n.Ready()
			// next[3] is 5 (the no-op sits at 5), so the rejected prev is 4.
			n.Step(Message{Type: MsgAppResp, From: 3, To: 1, Term: n.Status().Term, Reject: true,
				PrevLogIndex: 4, ConflictIndex: tc.conflictIndex, ConflictTerm: tc.conflictTerm})
			if n.next[3] != tc.wantNext {
				t.Fatalf("next = %d, want %d", n.next[3], tc.wantNext)
			}
			retry := onlyMsg(t, n)
			if retry.PrevLogIndex != tc.wantNext-1 {
				t.Fatalf("retry prev = %d, want %d", retry.PrevLogIndex, tc.wantNext-1)
			}
		})
	}
}

func TestStaleRejectIsIgnored(t *testing.T) {
	n := leaderNode(t)
	before := n.next[2]
	n.Step(Message{Type: MsgAppResp, From: 2, To: 1, Term: 1, Reject: true, PrevLogIndex: 7, ConflictIndex: 1})
	if n.next[2] != before {
		t.Fatalf("stale reject moved next from %d to %d", before, n.next[2])
	}
}

func TestLeaderRepairsDivergentFollowers(t *testing.T) {
	c := newTestCluster(t,
		[]uint64{1, 1, 2},    // leader-to-be
		[]uint64{1, 1},       // behind
		[]uint64{1, 3, 3, 3}, // diverged: an isolated old leader's entries
	)
	c.nodes[1].term = 4 // node 1 has already seen term 4
	c.elect(1)
	c.nodes[1].Tick() // one more heartbeat round settles every follower
	c.deliver()

	want := logTerms(c.nodes[1])
	for id, n := range c.nodes {
		if got := logTerms(n); !equalTerms(got, want) {
			t.Errorf("node %d log = %v, want %v", id, got, want)
		}
	}
}

func TestLeaderCommitsOnceMajorityHasEntry(t *testing.T) {
	c := newTestCluster(t, nil, nil, nil, nil, nil)
	n := c.nodes[1]
	tickUntil(t, n, 100, func() bool { return n.Status().Role == Candidate })
	n.Step(voteResp(2, 1, false))
	n.Step(voteResp(3, 1, false))
	n.Ready()

	ack := func(from NodeID) {
		n.Step(Message{Type: MsgAppResp, From: from, To: 1, Term: 1, MatchIndex: 1})
	}
	ack(2)
	if n.Status().Commit != 0 {
		t.Fatal("committed with 2/5 replicas")
	}
	ack(3)
	if n.Status().Commit != 1 {
		t.Fatalf("commit = %d, want 1 with 3/5 replicas", n.Status().Commit)
	}
}

// TestFigure8 checks the rule from Figure 8 of the Raft paper: a leader
// must not commit an entry from an earlier term just because a majority
// stores it, because a later leader could still overwrite it.
func TestFigure8(t *testing.T) {
	// Node 1 holds an entry from term 2 that never committed, and has since
	// seen term 3.
	c := newTestCluster(t, []uint64{1, 2}, []uint64{1}, []uint64{1})
	n := c.nodes[1]
	n.term = 3
	tickUntil(t, n, 100, func() bool { return n.Status().Role == Candidate })
	n.Step(voteResp(2, 4, false)) // leader of term 4; no-op at index 3
	n.Ready()

	// Followers 2 and 3 now store index 2 (term 2) but not the no-op.
	n.Step(Message{Type: MsgAppResp, From: 2, To: 1, Term: 4, MatchIndex: 2})
	n.Step(Message{Type: MsgAppResp, From: 3, To: 1, Term: 4, MatchIndex: 2})
	if n.Status().Commit != 0 {
		t.Fatalf("committed term-2 entry by counting replicas: commit = %d", n.Status().Commit)
	}
	// Once the term-4 no-op reaches a majority, everything up to it commits.
	n.Step(Message{Type: MsgAppResp, From: 2, To: 1, Term: 4, MatchIndex: 3})
	if n.Status().Commit != 3 {
		t.Fatalf("commit = %d, want 3", n.Status().Commit)
	}
}

func TestSingleNodeCommitsImmediately(t *testing.T) {
	n := campaignNode(t, 1)
	n.Propose("x")
	if n.Status().Commit != 2 {
		t.Fatalf("commit = %d, want 2", n.Status().Commit)
	}
}

func TestFollowersLearnCommitFromLeader(t *testing.T) {
	c := newTestCluster(t, nil, nil, nil)
	c.elect(1)
	c.nodes[1].Propose("x")
	c.deliver()
	c.nodes[1].Tick() // heartbeat carries the new commit index
	c.deliver()
	for id, n := range c.nodes {
		if n.Status().Commit != 2 {
			t.Errorf("node %d commit = %d, want 2", id, n.Status().Commit)
		}
	}
}

func TestProgressOnlyOnLeader(t *testing.T) {
	n := leaderNode(t)
	next, match := n.Progress()
	if next[2] != 1 || match[1] != 1 {
		t.Fatalf("next=%v match=%v", next, match)
	}
	next[2] = 99
	if n.next[2] == 99 {
		t.Fatal("Progress must return copies")
	}
	if next, _ := newTestNode(t, 1, 3).Progress(); next != nil {
		t.Fatal("follower reported progress")
	}
}
