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

// handleApp processes AppendEntries from the leader of our current term.
func (n *Node) handleApp(m Message) {
	if n.role == Leader {
		return // two leaders in one term is impossible (election safety)
	}
	n.becomeFollower(m.Term, m.From) // also resets the election timer
	n.send(Message{Type: MsgAppResp, To: m.From})
}
