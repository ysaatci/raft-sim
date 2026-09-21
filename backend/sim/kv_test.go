package sim

import (
	"maps"
	"testing"
)

func TestKVApply(t *testing.T) {
	kv := NewKV()
	for _, cmd := range []string{"set x 1", "", "set y hello world", "bogus", "set x 2", "set z"} {
		kv.Apply(cmd)
	}
	want := map[string]string{"x": "2", "y": "hello world"}
	if !maps.Equal(kv.Data(), want) {
		t.Fatalf("data = %v, want %v", kv.Data(), want)
	}
}

func TestClusterAppliesCommittedCommands(t *testing.T) {
	c := newTestCluster(t, 6)
	leader := waitForLeader(t, c)
	if err := c.Propose(leader, "set color blue"); err != nil {
		t.Fatal(err)
	}
	c.Advance(500)
	for id := range c.nodes {
		if v, _ := c.KV(nodeID(id)).Get("color"); v != "blue" {
			t.Errorf("node %d color = %q, want blue", id+1, v)
		}
	}
}

func TestRestartedNodeRebuildsKV(t *testing.T) {
	c := newTestCluster(t, 7)
	leader := waitForLeader(t, c)
	c.Propose(leader, "set a 1")
	c.Advance(500)
	follower := leader%5 + 1
	c.Crash(follower)
	c.Restart(follower)
	c.Advance(500) // learns the commit index from the next heartbeat
	if v, _ := c.KV(follower).Get("a"); v != "1" {
		t.Fatalf("restarted node a = %q, want 1", v)
	}
}
