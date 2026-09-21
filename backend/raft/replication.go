package raft

// tickLeader drives the heartbeat timer. Heartbeats are empty
// AppendEntries that keep followers from starting elections.
func (n *Node) tickLeader() {
	n.heartbeatElapsed++
	if n.heartbeatElapsed >= n.heartbeatTick {
		n.heartbeatElapsed = 0
		n.broadcastAppend()
	}
}

func (n *Node) broadcastAppend() {
	for _, p := range n.peers {
		if p != n.id {
			n.sendAppend(p)
		}
	}
}

// sendAppend sends peer `to` every entry from its nextIndex onwards. With
// nothing new to send it doubles as a heartbeat.
func (n *Node) sendAppend(to NodeID) {
	prev := n.next[to] - 1
	prevTerm, _ := n.log.term(prev)
	n.send(Message{
		Type:         MsgApp,
		To:           to,
		PrevLogIndex: prev,
		PrevLogTerm:  prevTerm,
		Entries:      n.log.slice(prev+1, n.log.lastIndex()+1),
		Commit:       n.commit,
	})
}

// handleAppResp updates a follower's progress, or backs its nextIndex up
// after a failed consistency check and retries.
func (n *Node) handleAppResp(m Message) {
	if n.role != Leader {
		return
	}
	from := m.From
	if !m.Reject {
		if m.MatchIndex > n.match[from] {
			n.match[from] = m.MatchIndex
		}
		n.next[from] = max(n.next[from], n.match[from]+1)
		n.maybeCommit()
		if n.next[from] <= n.log.lastIndex() {
			n.sendAppend(from) // still behind: keep going
		}
		return
	}

	prev := n.next[from] - 1
	if m.PrevLogIndex != prev {
		return // answer to an older request; its hint is outdated
	}
	next := m.ConflictIndex
	if m.ConflictTerm != 0 {
		// If we have entries from the follower's conflicting term, resume
		// right after our last one; otherwise skip the whole term.
		for i := n.log.lastIndex(); i > 0; i-- {
			if t, _ := n.log.term(i); t == m.ConflictTerm {
				next = i + 1
				break
			}
		}
	}
	// Always make progress backwards, but never below what is known to match.
	n.next[from] = max(min(next, prev), n.match[from]+1)
	n.sendAppend(from)
}

// handleApp processes AppendEntries from the leader of our current term
// (Raft §5.3, receiver implementation).
func (n *Node) handleApp(m Message) {
	if n.role == Leader {
		return // two leaders in one term is impossible (election safety)
	}
	n.becomeFollower(m.Term, m.From) // also resets the election timer

	// Consistency check: we must hold the entry preceding the new ones.
	prevTerm, ok := n.log.term(m.PrevLogIndex)
	if !ok {
		n.rejectApp(m, n.log.lastIndex()+1, 0)
		return
	}
	if prevTerm != m.PrevLogTerm {
		// Point the leader at the first index of the conflicting term so it
		// can skip the whole term in one round trip.
		first := m.PrevLogIndex
		for first > 1 {
			if t, _ := n.log.term(first - 1); t != prevTerm {
				break
			}
			first--
		}
		n.rejectApp(m, first, prevTerm)
		return
	}

	// Append new entries, truncating only on a real conflict: a delayed or
	// duplicated message must not erase entries we already hold.
	for i, e := range m.Entries {
		if t, ok := n.log.term(e.Index); ok {
			if t == e.Term {
				continue
			}
			if e.Index <= n.commit {
				panic("raft: conflicting entry below commit index")
			}
			n.log.truncateFrom(e.Index)
			n.emit(Event{Type: EventLogTruncated, Index: e.Index})
		}
		n.log.append(m.Entries[i:]...)
		must(n.storage.Append(m.Entries[i:]))
		break
	}

	lastNew := m.PrevLogIndex + uint64(len(m.Entries))
	if c := min(m.Commit, lastNew); c > n.commit {
		n.setCommit(c)
	}
	n.send(Message{Type: MsgAppResp, To: m.From, MatchIndex: lastNew})
}

func (n *Node) rejectApp(m Message, conflictIndex, conflictTerm uint64) {
	n.send(Message{
		Type:          MsgAppResp,
		To:            m.From,
		Reject:        true,
		PrevLogIndex:  m.PrevLogIndex,
		ConflictIndex: conflictIndex,
		ConflictTerm:  conflictTerm,
	})
}

// maybeCommit advances the commit index to the highest entry stored on a
// majority, but only if that entry is from the current term. Entries from
// earlier terms are committed indirectly, by committing a later entry on
// top of them; counting replicas of old entries is unsafe (Raft §5.4.2,
// Figure 8).
func (n *Node) maybeCommit() {
	for i := n.log.lastIndex(); i > n.commit; i-- {
		if t, _ := n.log.term(i); t != n.term {
			return // terms only decrease further back
		}
		replicas := 0
		for _, p := range n.peers {
			if n.match[p] >= i {
				replicas++
			}
		}
		if replicas >= n.quorum() {
			n.setCommit(i)
			return
		}
	}
}
