package sim

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ysaatci/raft-sim/backend/raft"
)

func newTestSim(t *testing.T, seed uint64) *Sim {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Seed = seed
	s, err := NewSim(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func stateJSON(t *testing.T, s *Sim) string {
	t.Helper()
	b, err := json.Marshal(s.Cluster().State())
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// eventfulRun drives a sim through crashes, partitions and writes, and
// returns snapshots of its state at t=1500 and t=3000.
func eventfulRun(t *testing.T, s *Sim) (mid, end string) {
	t.Helper()
	s.Advance(500)
	leader := s.Cluster().Leader()
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(s.Do(Action{Kind: ActPropose, Node: leader, Data: "set a 1"}))
	s.Advance(300)
	must(s.Do(Action{Kind: ActCrash, Node: leader}))
	s.Advance(400)
	must(s.Do(Action{Kind: ActPartition, Groups: [][]raft.NodeID{{1, 2}, {3, 4, 5}}}))
	s.Advance(300)
	mid = stateJSON(t, s) // t=1500
	must(s.Do(Action{Kind: ActHeal}))
	must(s.Do(Action{Kind: ActRestart, Node: leader}))
	lossy := LinkConfig{LatencyMs: 30, JitterMs: 10, DropRate: 0.1}
	must(s.Do(Action{Kind: ActSetNetwork, Link: &lossy}))
	s.Advance(1500)
	return mid, stateJSON(t, s)
}

func TestSeekBackwardReproducesState(t *testing.T) {
	s := newTestSim(t, 30)
	mid, end := eventfulRun(t, s)
	if err := s.Seek(1500); err != nil {
		t.Fatal(err)
	}
	if got := stateJSON(t, s); got != mid {
		t.Fatal("state after rewinding to t=1500 differs from the original run")
	}
	s.Advance(1500) // replays the recorded heal/restart/network actions
	if got := stateJSON(t, s); got != end {
		t.Fatal("state after replaying forward to t=3000 differs from the original run")
	}
}

func TestSeekKeepsEventHistoryConsistent(t *testing.T) {
	s := newTestSim(t, 31)
	eventfulRun(t, s)
	full := append([]TimedEvent(nil), s.History()...)
	s.Seek(1000)
	for _, e := range s.History() {
		if e.Time > 1000 {
			t.Fatalf("event from t=%d present after rewinding to 1000", e.Time)
		}
	}
	s.Seek(3000)
	if !reflect.DeepEqual(s.History(), full) {
		t.Fatal("history after replay differs from the original")
	}
}

func TestActionAfterRewindStartsNewBranch(t *testing.T) {
	s := newTestSim(t, 32)
	eventfulRun(t, s)
	recorded := len(s.Actions())
	s.Seek(1000)
	if err := s.Do(Action{Kind: ActHeal}); err != nil {
		t.Fatal(err)
	}
	if n := len(s.Actions()); n >= recorded {
		t.Fatalf("%d actions after branching, want fewer than %d", n, recorded)
	}
	last := s.Actions()[len(s.Actions())-1]
	if last.Kind != ActHeal || last.At != 1000 {
		t.Fatalf("last action = %+v, want heal at 1000", last)
	}
}

func TestFailedActionIsNotRecorded(t *testing.T) {
	s := newTestSim(t, 33)
	if err := s.Do(Action{Kind: ActCrash, Node: 9}); err == nil {
		t.Fatal("crash of a missing node succeeded")
	}
	if err := s.Do(Action{Kind: "explode"}); err == nil {
		t.Fatal("unknown action succeeded")
	}
	if len(s.Actions()) != 0 {
		t.Fatalf("failed actions recorded: %+v", s.Actions())
	}
}

func TestActionJSONRoundTrip(t *testing.T) {
	a := Action{At: 5, Kind: ActPartition, Groups: [][]raft.NodeID{{1}, {2, 3}}}
	b, _ := json.Marshal(a)
	var got Action
	if err := json.Unmarshal(b, &got); err != nil || !reflect.DeepEqual(got, a) {
		t.Fatalf("round trip = %+v, %v; json %s", got, err, b)
	}
}

func TestReplayReproducesARun(t *testing.T) {
	s := newTestSim(t, 34)
	_, end := eventfulRun(t, s)

	r, err := Replay(s.cfg, s.Actions())
	if err != nil {
		t.Fatal(err)
	}
	if r.Now() != 0 {
		t.Fatalf("replay starts at t=%d", r.Now())
	}
	r.Advance(3000)
	if stateJSON(t, r) != end {
		t.Fatal("replaying the recorded actions gave a different run")
	}
}

func TestReplayRejectsUnorderedActions(t *testing.T) {
	cfg := DefaultConfig()
	if _, err := Replay(cfg, []Action{{At: 50, Kind: ActHeal}, {At: 10, Kind: ActHeal}}); err == nil {
		t.Fatal("out-of-order actions accepted")
	}
	if _, err := Replay(cfg, []Action{{At: -1, Kind: ActHeal}}); err == nil {
		t.Fatal("negative time accepted")
	}
}
