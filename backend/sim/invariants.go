package sim

import (
	"fmt"

	"github.com/ysaatci/raft-sim/backend/raft"
)

// The safety properties Raft guarantees (Raft paper, Figure 3).
const (
	ElectionSafety     = "Election Safety"      // at most one leader per term
	LogMatching        = "Log Matching"         // same index and term => identical prefix
	LeaderCompleteness = "Leader Completeness"  // committed entries are in every later leader's log
	StateMachineSafety = "State Machine Safety" // no two nodes apply different entries at an index
)

// Violation records a broken safety property.
type Violation struct {
	Time      Time   `json:"time"`
	Invariant string `json:"invariant"`
	Detail    string `json:"detail"`
}

type commitRecord struct {
	entry raft.Entry
	term  uint64 // term of the node that first reported it committed
}

type leaderView struct {
	id      raft.NodeID
	term    uint64
	entries []raft.Entry
}

// checker watches a running cluster and records every safety violation.
// It is fed from the cluster's Ready processing, so it sees every state
// change as it happens rather than sampling.
type checker struct {
	leaders       map[uint64]raft.NodeID  // term -> leader
	committed     map[uint64]commitRecord // index -> first committed entry seen
	checkedCommit map[raft.NodeID]uint64  // per node: commit prefix already recorded
	applied       map[uint64]raft.Entry   // index -> first applied entry
	violations    []Violation
	reported      map[string]bool // dedupes violations found repeatedly
}

func newChecker() *checker {
	return &checker{
		leaders:       map[uint64]raft.NodeID{},
		committed:     map[uint64]commitRecord{},
		checkedCommit: map[raft.NodeID]uint64{},
		applied:       map[uint64]raft.Entry{},
		reported:      map[string]bool{},
	}
}

func (k *checker) fail(now Time, invariant, format string, args ...any) {
	detail := fmt.Sprintf(format, args...)
	if k.reported[invariant+detail] {
		return
	}
	k.reported[invariant+detail] = true
	k.violations = append(k.violations, Violation{Time: now, Invariant: invariant, Detail: detail})
}

// onLeader is called when node id becomes leader of term with the given log.
func (k *checker) onLeader(now Time, id raft.NodeID, term uint64, entries []raft.Entry) {
	if prev, ok := k.leaders[term]; ok && prev != id {
		k.fail(now, ElectionSafety, "nodes %d and %d both led term %d", prev, id, term)
	}
	k.leaders[term] = id
	for _, rec := range k.committed {
		if rec.term < term {
			k.checkHas(now, leaderView{id, term, entries}, rec.entry)
		}
	}
}

// onCommit is called when node id, in term, advances its commit index to
// commit. It records the newly committed entries and checks that every
// leader of a later term holds them.
func (k *checker) onCommit(now Time, id raft.NodeID, term uint64, entries []raft.Entry, commit uint64, leaders []leaderView) {
	for i := k.checkedCommit[id] + 1; i <= commit; i++ {
		e := entries[i-1]
		rec, ok := k.committed[i]
		if !ok {
			k.committed[i] = commitRecord{entry: e, term: term}
			for _, l := range leaders {
				if l.term > term {
					k.checkHas(now, l, e)
				}
			}
			continue
		}
		if rec.entry != e {
			k.fail(now, StateMachineSafety, "node %d committed %s at index %d, but %s was committed there before",
				id, fmtEntry(e), i, fmtEntry(rec.entry))
		}
	}
	k.checkedCommit[id] = max(k.checkedCommit[id], commit)
}

func (k *checker) checkHas(now Time, l leaderView, e raft.Entry) {
	if int(e.Index) > len(l.entries) || l.entries[e.Index-1] != e {
		k.fail(now, LeaderCompleteness, "leader %d of term %d is missing committed entry %s at index %d",
			l.id, l.term, fmtEntry(e), e.Index)
	}
}

// onApply is called when node id applies e to its state machine.
func (k *checker) onApply(now Time, id raft.NodeID, e raft.Entry) {
	if prev, ok := k.applied[e.Index]; !ok {
		k.applied[e.Index] = e
	} else if prev != e {
		k.fail(now, StateMachineSafety, "node %d applied %s at index %d, another node applied %s",
			id, fmtEntry(e), e.Index, fmtEntry(prev))
	}
}

// checkLogs verifies Log Matching across every pair of logs.
func (k *checker) checkLogs(now Time, logs map[raft.NodeID][]raft.Entry, ids []raft.NodeID) {
	for i, a := range ids {
		for _, b := range ids[i+1:] {
			la, lb := logs[a], logs[b]
			// Find the highest index where both logs have the same term;
			// everything up to it must be identical.
			match := -1
			for j := min(len(la), len(lb)) - 1; j >= 0; j-- {
				if la[j].Term == lb[j].Term {
					match = j
					break
				}
			}
			for j := 0; j <= match; j++ {
				if la[j] != lb[j] {
					k.fail(now, LogMatching, "nodes %d and %d agree at index %d (term %d) but differ at index %d",
						a, b, match+1, la[match].Term, j+1)
					break
				}
			}
		}
	}
}

func fmtEntry(e raft.Entry) string { return fmt.Sprintf("{term %d, %q}", e.Term, e.Data) }
