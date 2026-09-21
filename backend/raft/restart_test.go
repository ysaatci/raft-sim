package raft

import "testing"

// restart simulates a crash: node id is rebuilt from its storage alone.
func (c *testCluster) restart(id NodeID) *Node {
	c.t.Helper()
	n := newTestNodeWithStorage(c.t, id, len(c.nodes), c.storages[id])
	c.nodes[id] = n
	return n
}

func TestRestartRestoresPersistentState(t *testing.T) {
	c := newTestCluster(t, nil, nil, nil)
	c.elect(1)
	c.nodes[1].Propose("set x 1")
	c.deliver()
	c.nodes[1].Tick()
	c.deliver()
	before := c.nodes[2].Status()

	after := c.restart(2).Status()
	if after.Term != before.Term || after.VotedFor != before.VotedFor || after.Commit != before.Commit {
		t.Fatalf("hard state lost: before %+v, after %+v", before, after)
	}
	if after.LastIndex != before.LastIndex || after.LastTerm != before.LastTerm {
		t.Fatalf("log lost: before %+v, after %+v", before, after)
	}
	if after.Role != Follower || after.Leader != None || after.Applied != 0 {
		t.Fatalf("volatile state not reset: %+v", after)
	}
}

func TestRestartReplaysCommittedEntries(t *testing.T) {
	c := newTestCluster(t, nil, nil, nil)
	c.elect(1)
	c.nodes[1].Propose("set x 1")
	c.deliver()

	// The state machine is volatile, so a restarted node replays the
	// committed prefix of its log to rebuild it.
	ents := c.restart(1).Ready().CommittedEntries
	if len(ents) != 2 || ents[1].Data != "set x 1" {
		t.Fatalf("replayed %+v, want no-op then set x 1", ents)
	}
}

func TestRestartDoesNotVoteTwiceInSameTerm(t *testing.T) {
	c := newTestCluster(t, nil, nil, nil)
	n := c.nodes[1]
	n.Step(voteReq(2, 1, 0, 0))
	n = c.restart(1)
	n.Step(voteReq(3, 1, 0, 0))
	if resp := onlyMsg(t, n); !resp.Reject {
		t.Fatal("restarted node voted twice in term 1")
	}
}

func TestRestartedLeaderComesBackAsFollower(t *testing.T) {
	c := newTestCluster(t, nil, nil, nil)
	c.elect(1)
	if st := c.restart(1).Status(); st.Role != Follower || st.Term != 1 {
		t.Fatalf("status = %+v, want follower in term 1", st)
	}
}
