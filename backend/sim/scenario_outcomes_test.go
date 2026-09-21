package sim

import (
	"reflect"
	"slices"
	"testing"

	"github.com/ysaatci/raft-sim/backend/raft"
)

// These tests hold each built-in scenario to what its narration claims.

func playScenario(t *testing.T, id string) *Sim {
	t.Helper()
	sc, ok := ScenarioByID(id)
	if !ok {
		t.Fatalf("no scenario %q", id)
	}
	s, err := Load(sc)
	if err != nil {
		t.Fatal(err)
	}
	s.Advance(sc.Duration)
	for _, v := range s.Cluster().Violations() {
		t.Errorf("t=%dms %s: %s", v.Time, v.Invariant, v.Detail)
	}
	return s
}

// statusAt rewinds to time t and returns node id's status.
func statusAt(t *testing.T, s *Sim, at Time, id raft.NodeID) raft.Status {
	t.Helper()
	s.Seek(at)
	st, ok := s.Cluster().Status(id)
	if !ok {
		t.Fatalf("node %d is crashed at t=%d", id, at)
	}
	return st
}

func events(s *Sim, typ raft.EventType) []TimedEvent {
	var out []TimedEvent
	for _, e := range s.History() {
		if e.Type == typ {
			out = append(out, e)
		}
	}
	return out
}

func assertKVEverywhere(t *testing.T, s *Sim, want map[string]string) {
	t.Helper()
	c := s.Cluster()
	for id := raft.NodeID(1); int(id) <= len(c.nodes); id++ {
		if got := c.KV(id).Data(); !reflect.DeepEqual(got, want) {
			t.Errorf("node %d state = %v, want %v", id, got, want)
		}
	}
}

func assertLogsIdentical(t *testing.T, s *Sim) {
	t.Helper()
	var first []raft.Entry
	for i, sn := range s.Cluster().running() {
		if i == 0 {
			first = sn.node.Entries()
		} else if !reflect.DeepEqual(sn.node.Entries(), first) {
			t.Errorf("node %d log differs from node %d", sn.id, s.Cluster().running()[0].id)
		}
	}
}

func TestScenarioNormalReplication(t *testing.T) {
	s := playScenario(t, "normal-replication")
	leaders := events(s, raft.EventBecameLeader)
	if len(leaders) != 1 || leaders[0].Time >= 400 {
		t.Fatalf("want exactly one leader, elected before 400ms; got %+v", leaders)
	}
	assertKVEverywhere(t, s, map[string]string{"color": "blue", "size": "large"})
	assertLogsIdentical(t, s)
}

func TestScenarioLeaderCrash(t *testing.T) {
	s := playScenario(t, "leader-crash")
	acts := s.Actions()
	old, newLeader := acts[1].Node, acts[2].Node
	if acts[1].Kind != ActCrash || acts[3].Kind != ActRestart || acts[3].Node != old {
		t.Fatalf("actions = %+v", acts)
	}
	if newLeader == old {
		t.Fatal("the crashed node led again")
	}
	elected := events(s, raft.EventBecameLeader)[1]
	if elected.Time <= 1000 || elected.Time > 1700 || elected.Node != newLeader {
		t.Fatalf("new leader %+v; want node %d elected between 1000 and 1700ms", elected, newLeader)
	}
	if st, _ := s.Cluster().Status(old); st.Role != raft.Follower || st.Leader != newLeader {
		t.Errorf("restarted old leader = %+v, want follower of %d", st, newLeader)
	}
	assertKVEverywhere(t, s, map[string]string{"x": "2"})
	assertLogsIdentical(t, s)
}

func TestScenarioSplitVote(t *testing.T) {
	s := playScenario(t, "split-vote")
	var term1 []raft.NodeID
	for _, e := range events(s, raft.EventElectionStarted) {
		if e.Term == 1 {
			if e.Time < 140 || e.Time > 170 {
				t.Errorf("term-1 election at %dms, narration says 140-170", e.Time)
			}
			term1 = append(term1, e.Node)
		}
	}
	if !reflect.DeepEqual(term1, []raft.NodeID{3, 2}) {
		t.Errorf("term-1 candidates = %v, want [3 2]", term1)
	}
	votes := map[raft.NodeID]raft.NodeID{}
	for _, e := range events(s, raft.EventVoteGranted) {
		if e.Term == 1 {
			votes[e.Node] = e.Peer
		}
	}
	if !reflect.DeepEqual(votes, map[raft.NodeID]raft.NodeID{1: 2, 4: 3}) {
		t.Errorf("term-1 votes = %v, want N1->N2 and N4->N3", votes)
	}
	first := events(s, raft.EventBecameLeader)[0]
	if first.Node != 2 || first.Term != 2 || first.Time > 370 {
		t.Errorf("first leader = %+v, want N2 in term 2 by 370ms", first)
	}
	if down := events(s, raft.EventSteppedDown); len(down) == 0 || down[0].Node != 3 {
		t.Errorf("stepped down = %+v, want N3", down)
	}
}

func TestScenarioMinorityPartition(t *testing.T) {
	s := playScenario(t, "minority-partition")
	acts := s.Actions()
	old := acts[0].Node
	side, stale, fresh := acts[1].Groups[0], acts[2], acts[3]
	if side[0] != old || stale.Node != old {
		t.Fatalf("the stale write must go to the isolated old leader: %+v", acts)
	}
	if slices.Contains(side, fresh.Node) {
		t.Fatalf("new leader %d is on the minority side %v", fresh.Node, side)
	}
	assertKVEverywhere(t, s, map[string]string{"x": "fresh"})
	assertLogsIdentical(t, s)

	var truncated bool
	for _, e := range events(s, raft.EventLogTruncated) {
		truncated = truncated || e.Node == old
	}
	if !truncated {
		t.Error("the old leader's uncommitted entry was never overwritten")
	}
	if st := statusAt(t, s, 2000, old); st.Role != raft.Leader {
		t.Errorf("at 2000ms the old leader should still believe it leads: %+v", st)
	}
	if st := statusAt(t, s, 2000, fresh.Node); st.Role != raft.Leader || st.Term <= statusAt(t, s, 2000, old).Term {
		t.Errorf("at 2000ms the new leader should lead a higher term: %+v", st)
	}
}

func TestScenarioDivergentLogs(t *testing.T) {
	s := playScenario(t, "divergent-logs")
	acts := s.Actions()
	old, follower := acts[1].Groups[0][0], acts[1].Groups[0][1]
	if acts[2].Node != old || acts[3].Node != old || acts[4].Kind != ActCrash || acts[4].Node != old {
		t.Fatalf("the two writes and the crash must hit the partitioned old leader: %+v", acts)
	}
	var repaired bool
	for _, e := range events(s, raft.EventLogTruncated) {
		repaired = repaired || (e.Node == follower && e.Time >= 1500)
	}
	if !repaired {
		t.Error("the diverged follower's log was never repaired after the heal")
	}
	assertLogsIdentical(t, s)
	c := s.Cluster()
	for _, sn := range c.running() {
		if kv := c.KV(sn.id).Data(); !reflect.DeepEqual(kv, map[string]string{"a": "1", "d": "4"}) {
			t.Errorf("node %d state = %v; the uncommitted b and c must be gone", sn.id, kv)
		}
	}
}

func TestScenarioPacketLoss(t *testing.T) {
	s := playScenario(t, "packet-loss")
	assertKVEverywhere(t, s, map[string]string{"k": "3"})
	assertLogsIdentical(t, s)
}

func TestScenarioCrashDuringReplication(t *testing.T) {
	s := playScenario(t, "crash-during-replication")
	acts := s.Actions()
	if acts[1].Kind != ActCrash || acts[1].Node != acts[0].Node || acts[1].At-acts[0].At != 5 {
		t.Fatalf("the leader must crash 5ms after the write: %+v", acts)
	}
	if st := statusAt(t, s, 604, acts[0].Node); st.Commit >= st.LastIndex {
		t.Errorf("the old leader committed the write before crashing: %+v", st)
	}
	s.Seek(2800)
	assertKVEverywhere(t, s, map[string]string{"x": "1"})
	assertLogsIdentical(t, s)
}
