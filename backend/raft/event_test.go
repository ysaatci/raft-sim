package raft

import "testing"

func eventTypes(evs []Event) []EventType {
	var out []EventType
	for _, e := range evs {
		out = append(out, e.Type)
	}
	return out
}

func hasEvent(evs []Event, want Event) bool {
	for _, e := range evs {
		if e == want {
			return true
		}
	}
	return false
}

func TestElectionEvents(t *testing.T) {
	c := newTestCluster(t, nil, nil, nil)
	n := c.nodes[1]
	tickUntil(t, n, 100, func() bool { return n.Status().Role == Candidate })
	if evs := n.Ready().Events; !hasEvent(evs, Event{Type: EventElectionStarted, Node: 1, Term: 1}) {
		t.Fatalf("events = %+v, want election-started", evs)
	}

	c.nodes[2].Step(voteReq(1, 1, 0, 0))
	if evs := c.nodes[2].Ready().Events; !hasEvent(evs, Event{Type: EventVoteGranted, Node: 2, Term: 1, Peer: 1}) {
		t.Fatalf("events = %+v, want vote-granted to 1", evs)
	}

	n.Step(voteResp(2, 1, false))
	if evs := n.Ready().Events; !hasEvent(evs, Event{Type: EventBecameLeader, Node: 1, Term: 1}) {
		t.Fatalf("events = %+v, want became-leader", evs)
	}

	n.Step(Message{Type: MsgApp, From: 3, To: 1, Term: 2})
	if evs := n.Ready().Events; !hasEvent(evs, Event{Type: EventSteppedDown, Node: 1, Term: 2}) {
		t.Fatalf("events = %+v, want stepped-down", evs)
	}
}

func TestReplicationEvents(t *testing.T) {
	n, _ := followerWithLog(t, 1, 2)
	n.Step(app(1, 1, 2, ent(2, 3)))
	evs := n.Ready().Events
	if !hasEvent(evs, Event{Type: EventLogTruncated, Node: 1, Term: 3, Index: 2}) {
		t.Errorf("events = %+v, want log-truncated at 2", evs)
	}
	if !hasEvent(evs, Event{Type: EventCommitAdvanced, Node: 1, Term: 3, Index: 2}) {
		t.Errorf("events = %+v, want commit-advanced to 2", evs)
	}
}

func TestReadyDrainsEvents(t *testing.T) {
	n := campaignNode(t, 3)
	if evs := n.Ready().Events; len(evs) != 0 {
		t.Fatalf("events not drained: %v", eventTypes(evs))
	}
}
