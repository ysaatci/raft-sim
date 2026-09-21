package sim

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ysaatci/raft-sim/backend/raft"
)

func TestStateDescribesCluster(t *testing.T) {
	c := newTestCluster(t, 20)
	leader := waitForLeader(t, c)
	c.Propose(leader, "set x 1")
	c.Advance(200)
	follower := leader%5 + 1
	c.Crash(follower)
	c.Advance(5)

	s := c.State()
	if s.Time != c.Now() || len(s.Nodes) != 5 || len(s.Links) != 20 {
		t.Fatalf("time=%d nodes=%d links=%d", s.Time, len(s.Nodes), len(s.Links))
	}
	lv := s.Nodes[leader-1]
	if lv.Role != raft.Leader || lv.State != NodeUp || lv.Next == nil || lv.KV["x"] != "1" {
		t.Fatalf("leader view = %+v", lv)
	}
	fv := s.Nodes[follower-1]
	if fv.State != NodeCrashed || fv.Term == 0 || len(fv.Log) != 2 || fv.LastIndex != 2 {
		t.Fatalf("crashed node view should show persisted state: %+v", fv)
	}
	if len(s.Flights) == 0 {
		t.Fatal("no messages in flight between heartbeats")
	}
	for i := 1; i < len(s.Flights); i++ {
		if s.Flights[i-1].ID >= s.Flights[i].ID {
			t.Fatal("flights not sorted by ID")
		}
	}
}

func TestStateShowsDroppedMessagesUntilDue(t *testing.T) {
	c := newTestCluster(t, 21)
	waitForLeader(t, c)
	c.SetNetwork(LinkConfig{LatencyMs: 50, DropRate: 1})
	c.Advance(60) // at least one heartbeat round is sent and lost

	var dropped []Flight
	for _, f := range c.State().Flights {
		if f.Dropped {
			dropped = append(dropped, f)
		}
	}
	if len(dropped) == 0 {
		t.Fatal("no dropped messages visible")
	}
	for _, f := range dropped {
		if f.DeliverAt < c.Now() {
			t.Fatalf("stale dropped message kept: %+v at t=%d", f, c.Now())
		}
	}
}

func TestStateJSON(t *testing.T) {
	c := newTestCluster(t, 22)
	waitForLeader(t, c)
	b, err := json.Marshal(c.State())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"role":"leader"`, `"state":"up"`, `"type":"AppendEntries"`, `"latencyMs":20`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("JSON missing %s", want)
		}
	}
}
