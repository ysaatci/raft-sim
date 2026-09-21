package sim

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ysaatci/raft-sim/backend/raft"
)

func TestTargetsResolveByRole(t *testing.T) {
	c := newTestCluster(t, 40)
	leader := waitForLeader(t, c)
	followers := others(5, leader)

	cases := map[string]raft.NodeID{"leader": leader, "follower:1": followers[0], "follower:4": followers[3], "node:3": 3}
	for tgt, want := range cases {
		if got, err := target(c, tgt); err != nil || got != want {
			t.Errorf("target(%q) = %d, %v; want %d", tgt, got, err, want)
		}
	}
	for _, bad := range []string{"follower:5", "follower:0", "follower:x", "node:x", "boss"} {
		if _, err := target(c, bad); err == nil {
			t.Errorf("target(%q) should fail", bad)
		}
	}
}

func TestFollowerTargetsSkipCrashedNodes(t *testing.T) {
	c := newTestCluster(t, 41)
	leader := waitForLeader(t, c)
	first := others(5, leader)[0]
	c.Crash(first)
	if got, _ := target(c, "follower:1"); got == first {
		t.Fatal("follower:1 resolved to a crashed node")
	}
}

func testScenario() Scenario {
	cfg := DefaultConfig()
	cfg.Seed = 42
	return Scenario{
		ID:       "test",
		Config:   cfg,
		Duration: 2000,
		Steps: []Step{
			{At: 0, Narration: "Nodes start as followers."},
			{At: 600, Narration: "Write to the leader.", Kind: ActPropose, Target: "leader", Data: "set x 1"},
			{At: 900, Narration: "Isolate the leader.", Kind: ActPartition, Side: []string{"leader", "follower:1"}},
			{At: 1500, Narration: "Heal.", Kind: ActHeal},
		},
	}
}

func TestLoadRecordsConcreteActionsAndRewinds(t *testing.T) {
	s, err := Load(testScenario())
	if err != nil {
		t.Fatal(err)
	}
	if s.Now() != 0 {
		t.Fatalf("loaded scenario at t=%d, want 0", s.Now())
	}
	acts := s.Actions()
	if len(acts) != 3 {
		t.Fatalf("recorded %d actions, want 3 (narration-only steps record nothing)", len(acts))
	}
	if acts[0].Kind != ActPropose || acts[0].Node == raft.None || acts[0].At != 600 {
		t.Errorf("propose = %+v", acts[0])
	}
	p := acts[1]
	if len(p.Groups) != 2 || len(p.Groups[0]) != 2 || len(p.Groups[1]) != 3 || p.Groups[0][0] != acts[0].Node {
		t.Errorf("partition = %+v, want [leader, follower] vs the other three", p)
	}
}

func TestLoadedScenarioReplaysTheScript(t *testing.T) {
	s, _ := Load(testScenario())
	s.Advance(2000)
	if v, _ := s.Cluster().KV(1).Get("x"); v != "1" {
		t.Fatalf("x = %q after replay, want 1", v)
	}
	if len(s.Cluster().Violations()) != 0 {
		t.Fatalf("violations: %+v", s.Cluster().Violations())
	}

	// Replaying twice gives the same run.
	again, _ := Load(testScenario())
	again.Advance(2000)
	if !reflect.DeepEqual(s.History(), again.History()) {
		t.Fatal("loading the same scenario twice gave different runs")
	}
}

func TestLoadRejectsBrokenScripts(t *testing.T) {
	sc := testScenario()
	sc.Steps = []Step{{At: 0, Kind: ActCrash, Target: "leader"}} // nobody leads yet
	if _, err := Load(sc); err == nil || !strings.Contains(err.Error(), "no leader") {
		t.Errorf("err = %v, want no leader", err)
	}
	sc.Steps = []Step{{At: 500}, {At: 100}}
	if _, err := Load(sc); err == nil || !strings.Contains(err.Error(), "back in time") {
		t.Errorf("err = %v, want back in time", err)
	}
}
