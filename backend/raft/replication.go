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

func (n *Node) sendAppend(to NodeID) {
	n.send(Message{Type: MsgApp, To: to})
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
		}
		n.log.append(m.Entries[i:]...)
		must(n.storage.Append(m.Entries[i:]))
		break
	}

	lastNew := m.PrevLogIndex + uint64(len(m.Entries))
	if c := min(m.Commit, lastNew); c > n.commit {
		n.commit = c
		n.persist()
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
