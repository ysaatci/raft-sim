package raft

// raftLog is the in-memory Raft log. Index 0 holds a sentinel entry with
// term 0, so every real entry sits at entries[i] for index i and
// "the entry before the first one" always exists.
type raftLog struct {
	entries []Entry
}

func newLog() *raftLog {
	return &raftLog{entries: []Entry{{}}}
}

func (l *raftLog) lastIndex() uint64 { return uint64(len(l.entries) - 1) }

func (l *raftLog) lastTerm() uint64 { return l.entries[len(l.entries)-1].Term }

// term returns the term of the entry at index i, and false if there is none.
func (l *raftLog) term(i uint64) (uint64, bool) {
	if i > l.lastIndex() {
		return 0, false
	}
	return l.entries[i].Term, true
}

// append adds entries to the end of the log. Their indices must continue
// the log without gaps.
func (l *raftLog) append(ents ...Entry) {
	for _, e := range ents {
		if e.Index != l.lastIndex()+1 {
			panic("raft: non-contiguous append")
		}
		l.entries = append(l.entries, e)
	}
}

// truncateFrom removes the entry at index i and everything after it.
func (l *raftLog) truncateFrom(i uint64) {
	if i == 0 {
		panic("raft: cannot truncate the sentinel entry")
	}
	if i <= l.lastIndex() {
		l.entries = l.entries[:i]
	}
}

// slice returns a copy of the entries in [lo, hi).
func (l *raftLog) slice(lo, hi uint64) []Entry {
	if lo >= hi {
		return nil
	}
	return append([]Entry(nil), l.entries[lo:hi]...)
}

// isUpToDate reports whether a log ending at (lastIndex, lastTerm) is at
// least as up-to-date as this one (Raft §5.4.1).
func (l *raftLog) isUpToDate(lastIndex, lastTerm uint64) bool {
	return lastTerm > l.lastTerm() || (lastTerm == l.lastTerm() && lastIndex >= l.lastIndex())
}
