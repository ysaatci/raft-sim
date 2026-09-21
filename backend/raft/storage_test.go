package raft

import (
	"reflect"
	"testing"
)

func TestMemoryStorageHardState(t *testing.T) {
	s := NewMemoryStorage()
	want := HardState{Term: 3, VotedFor: 2, Commit: 5}
	if err := s.SetHardState(want); err != nil {
		t.Fatal(err)
	}
	hs, _, _ := s.InitialState()
	if hs != want {
		t.Fatalf("hard state = %+v, want %+v", hs, want)
	}
}

func TestMemoryStorageAppendReplacesSuffix(t *testing.T) {
	s := NewMemoryStorage()
	s.Append([]Entry{{Term: 1, Index: 1}, {Term: 1, Index: 2}, {Term: 1, Index: 3}})
	s.Append([]Entry{{Term: 2, Index: 2}})
	_, ents, _ := s.InitialState()
	want := []Entry{{Term: 1, Index: 1}, {Term: 2, Index: 2}}
	if !reflect.DeepEqual(ents, want) {
		t.Fatalf("entries = %v, want %v", ents, want)
	}
}

func TestMemoryStorageInitialStateCopies(t *testing.T) {
	s := NewMemoryStorage()
	s.Append([]Entry{{Term: 1, Index: 1}})
	_, ents, _ := s.InitialState()
	ents[0].Term = 99
	_, ents, _ = s.InitialState()
	if ents[0].Term != 1 {
		t.Fatal("InitialState must return a copy")
	}
}

func TestMemoryStorageAppendGapPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	NewMemoryStorage().Append([]Entry{{Term: 1, Index: 2}})
}
