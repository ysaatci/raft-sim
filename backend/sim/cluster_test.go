package sim

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ysaatci/raft-sim/backend/raft"
)

func newTestCluster(t *testing.T, seed uint64) *Cluster {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Seed = seed
	c, err := NewCluster(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// waitForLeader advances until some node leads, failing after 5s.
func waitForLeader(t *testing.T, c *Cluster) raft.NodeID {
	t.Helper()
	for range 500 {
		c.Advance(10)
		if l := c.Leader(); l != raft.None {
			return l
		}
	}
	t.Fatal("no leader elected within 5s")
	return raft.None
}

func TestNewClusterValidatesConfig(t *testing.T) {
	for _, mutate := range []func(*Config){
		func(c *Config) { c.Size = 0 },
		func(c *Config) { c.Size = 10 },
		func(c *Config) { c.TickMs = 0 },
		func(c *Config) { c.HeartbeatTick = c.ElectionTick },
	} {
		cfg := DefaultConfig()
		mutate(&cfg)
		if _, err := NewCluster(cfg); err == nil {
			t.Errorf("config %+v accepted", cfg)
		}
	}
}

func TestClusterElectsOneLeader(t *testing.T) {
	c := newTestCluster(t, 1)
	leader := waitForLeader(t, c)
	c.Advance(1000)
	if c.Leader() != leader {
		t.Fatalf("leader changed from %d to %d on a healthy network", leader, c.Leader())
	}
	for id := raft.NodeID(1); id <= 5; id++ {
		if st, _ := c.Status(id); st.Leader != leader {
			t.Errorf("node %d thinks leader is %d, want %d", id, st.Leader, leader)
		}
	}
}

func TestCrashedLeaderIsReplaced(t *testing.T) {
	c := newTestCluster(t, 2)
	old := waitForLeader(t, c)
	oldTerm := mustStatus(t, c, old).Term
	c.Crash(old)
	if _, ok := c.Status(old); ok || c.NodeState(old) != NodeCrashed {
		t.Fatal("crashed node still reports status")
	}
	next := waitForLeader(t, c)
	if next == old || mustStatus(t, c, next).Term <= oldTerm {
		t.Fatalf("new leader %d in term %d, old was %d in term %d", next, mustStatus(t, c, next).Term, old, oldTerm)
	}

	c.Restart(old)
	c.Advance(500)
	if st := mustStatus(t, c, old); st.Role != raft.Follower || st.Leader != next {
		t.Fatalf("restarted node status %+v, want follower of %d", st, next)
	}
}

func TestPausedNodeQueuesMessages(t *testing.T) {
	c := newTestCluster(t, 3)
	leader := waitForLeader(t, c)
	follower := leader%5 + 1
	c.Pause(follower)
	c.Advance(200)
	if len(c.node(follower).inbox) == 0 {
		t.Fatal("paused node received nothing into its inbox")
	}
	c.Resume(follower)
	if len(c.node(follower).inbox) != 0 {
		t.Fatal("inbox not processed on resume")
	}
}

func TestProposeOnFollowerFails(t *testing.T) {
	c := newTestCluster(t, 4)
	leader := waitForLeader(t, c)
	var nle *raft.NotLeaderError
	if err := c.Propose(leader%5+1, "set x 1"); !errors.As(err, &nle) {
		t.Fatalf("err = %v, want NotLeaderError", err)
	}
	if err := c.Propose(leader, "set x 1"); err != nil {
		t.Fatal(err)
	}
}

func TestPartitionedMinorityCannotElect(t *testing.T) {
	c := newTestCluster(t, 5)
	waitForLeader(t, c)
	c.Partition([]raft.NodeID{1, 2}, []raft.NodeID{3, 4, 5})
	c.Advance(3000)
	for _, id := range []raft.NodeID{1, 2} {
		if mustStatus(t, c, id).Role == raft.Leader && mustStatus(t, c, id).Term > mustStatus(t, c, 3).Term {
			t.Errorf("minority node %d leads a newer term", id)
		}
	}
	if l := c.Leader(); l < 3 {
		t.Fatalf("leader %d is not in the majority", l)
	}
}

func TestClusterIsDeterministic(t *testing.T) {
	run := func() []TimedEvent {
		c := newTestCluster(t, 42)
		c.SetNetwork(LinkConfig{LatencyMs: 20, JitterMs: 15, DropRate: 0.1})
		c.Advance(2000)
		c.Crash(c.Leader())
		c.Advance(2000)
		return c.Events()
	}
	a, b := run(), run()
	if len(a) == 0 || !reflect.DeepEqual(a, b) {
		t.Fatalf("runs differ or are empty: %d vs %d events", len(a), len(b))
	}
}

func mustStatus(t *testing.T, c *Cluster, id raft.NodeID) raft.Status {
	t.Helper()
	st, ok := c.Status(id)
	if !ok {
		t.Fatalf("node %d is crashed", id)
	}
	return st
}
