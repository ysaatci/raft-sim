package sim

import (
	"testing"

	"github.com/ysaatci/raft-sim/backend/raft"
)

func entries(terms ...uint64) []raft.Entry {
	var out []raft.Entry
	for i, t := range terms {
		out = append(out, raft.Entry{Term: t, Index: uint64(i + 1)})
	}
	return out
}

func wantViolation(t *testing.T, k *checker, invariant string) {
	t.Helper()
	if len(k.violations) != 1 || k.violations[0].Invariant != invariant {
		t.Fatalf("violations = %+v, want one %s", k.violations, invariant)
	}
}

func TestCheckerElectionSafety(t *testing.T) {
	k := newChecker()
	k.onLeader(0, 1, 3, nil)
	k.onLeader(0, 1, 3, nil) // same leader again is fine
	if len(k.violations) != 0 {
		t.Fatal("re-reporting the same leader flagged")
	}
	k.onLeader(0, 2, 3, nil)
	wantViolation(t, k, ElectionSafety)
}

func TestCheckerLogMatching(t *testing.T) {
	k := newChecker()
	ids := []raft.NodeID{1, 2}
	good := map[raft.NodeID][]raft.Entry{1: entries(1, 1, 2), 2: entries(1, 1, 3, 3)}
	k.checkLogs(0, good, ids)
	if len(k.violations) != 0 {
		t.Fatalf("diverging suffix flagged: %+v", k.violations)
	}
	bad := map[raft.NodeID][]raft.Entry{1: entries(1, 1, 2), 2: entries(1, 2, 2)}
	k.checkLogs(0, bad, ids)
	k.checkLogs(1, bad, ids) // reported once, not per check
	wantViolation(t, k, LogMatching)
}

func TestCheckerLeaderCompletenessOnElection(t *testing.T) {
	k := newChecker()
	k.onCommit(0, 1, 2, entries(1, 2), 2, nil)
	k.onLeader(0, 3, 3, entries(1)) // term 3 leader lacks index 2
	wantViolation(t, k, LeaderCompleteness)
}

func TestCheckerLeaderCompletenessOnLateCommit(t *testing.T) {
	k := newChecker()
	// A leader of term 4 exists when an old leader of term 3 commits.
	leaders := []leaderView{{id: 2, term: 4, entries: entries(1)}}
	k.onCommit(0, 1, 3, entries(1, 3), 2, leaders)
	wantViolation(t, k, LeaderCompleteness)
}

func TestCheckerIgnoresOlderLeaders(t *testing.T) {
	k := newChecker()
	k.onCommit(0, 1, 3, entries(1, 3), 2, nil)
	k.onLeader(0, 2, 2, entries(1)) // a stale leader of an older term
	if len(k.violations) != 0 {
		t.Fatalf("stale leader flagged: %+v", k.violations)
	}
}

func TestCheckerStateMachineSafety(t *testing.T) {
	k := newChecker()
	k.onApply(0, 1, raft.Entry{Term: 1, Index: 1, Data: "set x 1"})
	k.onApply(0, 2, raft.Entry{Term: 1, Index: 1, Data: "set x 1"})
	if len(k.violations) != 0 {
		t.Fatal("identical applies flagged")
	}
	k.onApply(0, 3, raft.Entry{Term: 2, Index: 1, Data: "set x 2"})
	wantViolation(t, k, StateMachineSafety)
}

func TestCheckerConflictingCommits(t *testing.T) {
	k := newChecker()
	k.onCommit(0, 1, 1, entries(1), 1, nil)
	k.onCommit(0, 2, 2, entries(2), 1, nil)
	wantViolation(t, k, StateMachineSafety)
}

func TestHealthyClusterHasNoViolations(t *testing.T) {
	c := newTestCluster(t, 8)
	for i := range 20 {
		c.Advance(100)
		if l := c.Leader(); l != raft.None {
			c.Propose(l, "set k "+string(rune('a'+i)))
		}
	}
	if len(c.check.committed) < 10 {
		t.Fatalf("only %d entries committed; checker saw too little", len(c.check.committed))
	}
	if v := c.Violations(); len(v) != 0 {
		t.Fatalf("violations: %+v", v)
	}
}
