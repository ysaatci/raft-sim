package sim

import (
	"fmt"
	"testing"

	"github.com/ysaatci/raft-sim/backend/raft"
)

// others returns every node ID except the given ones.
func others(size int, except ...raft.NodeID) []raft.NodeID {
	var out []raft.NodeID
outer:
	for id := raft.NodeID(1); int(id) <= size; id++ {
		for _, e := range except {
			if id == e {
				continue outer
			}
		}
		out = append(out, id)
	}
	return out
}

func assertNoViolations(t *testing.T, c *Cluster) {
	t.Helper()
	for _, v := range c.Violations() {
		t.Errorf("t=%dms %s: %s", v.Time, v.Invariant, v.Detail)
	}
}

func assertAllKV(t *testing.T, c *Cluster, key, want string) {
	t.Helper()
	for id := raft.NodeID(1); int(id) <= len(c.nodes); id++ {
		if got, _ := c.KV(id).Get(key); got != want {
			t.Errorf("node %d: %s = %q, want %q", id, key, got, want)
		}
	}
}

func TestReplicationHappyPath(t *testing.T) {
	c := newTestCluster(t, 10)
	leader := waitForLeader(t, c)
	for i := range 10 {
		if err := c.Propose(leader, fmt.Sprintf("set k%d v%d", i, i)); err != nil {
			t.Fatal(err)
		}
		c.Advance(20)
	}
	c.Advance(500)
	for i := range 10 {
		assertAllKV(t, c, fmt.Sprintf("k%d", i), fmt.Sprintf("v%d", i))
	}
	assertNoViolations(t, c)
}

func TestCommittedDataSurvivesLeaderCrash(t *testing.T) {
	c := newTestCluster(t, 11)
	old := waitForLeader(t, c)
	c.Propose(old, "set x 1")
	c.Advance(300) // committed and replicated
	c.Crash(old)

	next := waitForLeader(t, c)
	c.Propose(next, "set y 2")
	c.Advance(300)
	c.Restart(old)
	c.Advance(500)

	assertAllKV(t, c, "x", "1")
	assertAllKV(t, c, "y", "2")
	assertNoViolations(t, c)
}

func TestMinorityLeaderWritesAreDiscarded(t *testing.T) {
	c := newTestCluster(t, 12)
	old := waitForLeader(t, c)
	c.Propose(old, "set x original")
	c.Advance(300)

	// Isolate the leader with one follower; the other three form a majority.
	minority := []raft.NodeID{old, old%5 + 1}
	majority := others(5, minority...)
	c.Partition(minority, majority)

	if err := c.Propose(old, "set x stale"); err != nil {
		t.Fatalf("old leader should still accept writes it cannot commit: %v", err)
	}
	c.Advance(1000)
	if st := mustStatus(t, c, old); st.Role != raft.Leader {
		t.Fatalf("old leader stepped down without hearing a newer term: %+v", st)
	}
	if v, _ := c.KV(old).Get("x"); v != "original" {
		t.Fatalf("minority committed a write: x = %q", v)
	}

	next := c.Leader()
	if next == old || next == minority[1] {
		t.Fatalf("leader %d is not in the majority", next)
	}
	c.Propose(next, "set x fresh")
	c.Advance(300)

	c.Heal()
	c.Advance(1000)
	assertAllKV(t, c, "x", "fresh")
	if c.Leader() != next {
		t.Fatalf("leader after heal = %d, want %d", c.Leader(), next)
	}
	sawTruncate := false
	for _, e := range c.Events() {
		if e.Type == raft.EventLogTruncated && e.Node == old {
			sawTruncate = true
		}
	}
	if !sawTruncate {
		t.Error("old leader's uncommitted entry was never truncated")
	}
	assertNoViolations(t, c)
}

func TestLaggingFollowerCatchesUp(t *testing.T) {
	c := newTestCluster(t, 13)
	leader := waitForLeader(t, c)
	lagger := leader%5 + 1
	c.Crash(lagger)
	for i := range 50 {
		c.Propose(leader, fmt.Sprintf("set n %d", i))
		c.Advance(10)
	}
	c.Restart(lagger)
	c.Advance(1000)
	leaderSt, lagSt := mustStatus(t, c, leader), mustStatus(t, c, lagger)
	if lagSt.LastIndex != leaderSt.LastIndex || lagSt.Commit != leaderSt.Commit {
		t.Fatalf("lagger %+v did not catch up with leader %+v", lagSt, leaderSt)
	}
	assertAllKV(t, c, "n", "49")
	assertNoViolations(t, c)
}

func TestClusterSurvivesLossyNetwork(t *testing.T) {
	c := newTestCluster(t, 14)
	c.SetNetwork(LinkConfig{LatencyMs: 20, JitterMs: 15, DropRate: 0.2})
	committed := 0
	for i := range 100 {
		c.Advance(50)
		if l := c.Leader(); l != raft.None && c.Propose(l, fmt.Sprintf("set k %d", i)) == nil {
			committed++
		}
	}
	c.SetNetwork(LinkConfig{LatencyMs: 20})
	c.Advance(2000)
	if committed < 50 {
		t.Fatalf("only %d/100 proposals accepted on a 20%% lossy network", committed)
	}
	// All nodes converge on the same value, whichever write won.
	want, _ := c.KV(c.Leader()).Get("k")
	assertAllKV(t, c, "k", want)
	assertNoViolations(t, c)
}
