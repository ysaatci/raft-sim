package raft

import (
	"reflect"
	"testing"
)

// logWithTerms builds a log whose entry i+1 has term terms[i].
func logWithTerms(terms ...uint64) *raftLog {
	l := newLog()
	for i, t := range terms {
		l.append(Entry{Term: t, Index: uint64(i + 1)})
	}
	return l
}

func TestLogEmpty(t *testing.T) {
	l := newLog()
	if l.lastIndex() != 0 || l.lastTerm() != 0 {
		t.Fatalf("empty log: lastIndex=%d lastTerm=%d", l.lastIndex(), l.lastTerm())
	}
	if term, ok := l.term(0); !ok || term != 0 {
		t.Fatalf("sentinel term = %d, %v", term, ok)
	}
	if _, ok := l.term(1); ok {
		t.Fatal("term(1) on empty log should not exist")
	}
}

func TestLogAppendAndTerm(t *testing.T) {
	l := logWithTerms(1, 1, 2)
	if l.lastIndex() != 3 || l.lastTerm() != 2 {
		t.Fatalf("lastIndex=%d lastTerm=%d", l.lastIndex(), l.lastTerm())
	}
	for i, want := range []uint64{0, 1, 1, 2} {
		if got, ok := l.term(uint64(i)); !ok || got != want {
			t.Errorf("term(%d) = %d, %v; want %d", i, got, ok, want)
		}
	}
}

func TestLogAppendNonContiguousPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	newLog().append(Entry{Term: 1, Index: 2})
}

func TestLogTruncateFrom(t *testing.T) {
	l := logWithTerms(1, 1, 2, 3)
	l.truncateFrom(3)
	if l.lastIndex() != 2 || l.lastTerm() != 1 {
		t.Fatalf("after truncate: lastIndex=%d lastTerm=%d", l.lastIndex(), l.lastTerm())
	}
	l.truncateFrom(10) // beyond the end: no-op
	if l.lastIndex() != 2 {
		t.Fatalf("truncate beyond end changed log: lastIndex=%d", l.lastIndex())
	}
}

func TestLogSliceCopies(t *testing.T) {
	l := logWithTerms(1, 2, 3)
	got := l.slice(2, 4)
	want := []Entry{{Term: 2, Index: 2}, {Term: 3, Index: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("slice = %v, want %v", got, want)
	}
	got[0].Term = 99
	if term, _ := l.term(2); term != 2 {
		t.Fatal("slice must return a copy")
	}
	if l.slice(3, 3) != nil {
		t.Fatal("empty range should be nil")
	}
}

func TestLogIsUpToDate(t *testing.T) {
	l := logWithTerms(1, 2, 2) // last = (3, 2)
	cases := []struct {
		index, term uint64
		want        bool
	}{
		{3, 2, true},  // identical
		{4, 2, true},  // same term, longer
		{2, 2, false}, // same term, shorter
		{1, 3, true},  // higher term wins regardless of length
		{9, 1, false}, // lower term loses regardless of length
	}
	for _, c := range cases {
		if got := l.isUpToDate(c.index, c.term); got != c.want {
			t.Errorf("isUpToDate(%d, %d) = %v, want %v", c.index, c.term, got, c.want)
		}
	}
}
