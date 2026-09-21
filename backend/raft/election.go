package raft

// campaign starts an election: bump the term, vote for ourselves and ask
// every other node for its vote.
func (n *Node) campaign() {
	n.term++
	n.role = Candidate
	n.votedFor = n.id
	n.leader = None
	n.persist()
	n.resetTimers()
	n.votes = map[NodeID]bool{n.id: true}

	for _, p := range n.peers {
		if p == n.id {
			continue
		}
		n.send(Message{
			Type:         MsgVote,
			To:           p,
			LastLogIndex: n.log.lastIndex(),
			LastLogTerm:  n.log.lastTerm(),
		})
	}
}
