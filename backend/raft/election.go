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
	if n.quorum() == 1 { // single-node cluster
		n.becomeLeader()
		return
	}

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

// handleVote answers a RequestVote from a candidate in our current term.
// We grant at most one vote per term, and only to a candidate whose log is
// at least as up-to-date as ours (Raft §5.2, §5.4.1).
func (n *Node) handleVote(m Message) {
	canVote := n.votedFor == None || n.votedFor == m.From
	if !canVote || !n.log.isUpToDate(m.LastLogIndex, m.LastLogTerm) {
		n.send(Message{Type: MsgVoteResp, To: m.From, Reject: true})
		return
	}
	n.votedFor = m.From
	n.persist()
	n.resetTimers() // granting a vote defers our own candidacy
	n.send(Message{Type: MsgVoteResp, To: m.From})
}

// handleVoteResp tallies a vote from our current term and takes over as
// leader once a majority has granted it.
func (n *Node) handleVoteResp(m Message) {
	if n.role != Candidate {
		return // already won or stepped down; late responses are irrelevant
	}
	n.votes[m.From] = !m.Reject
	granted := 0
	for _, v := range n.votes {
		if v {
			granted++
		}
	}
	if granted >= n.quorum() {
		n.becomeLeader()
	}
}

func (n *Node) becomeLeader() {
	n.role = Leader
	n.leader = n.id
	n.votes = nil
	n.resetTimers()
	n.broadcastAppend() // assert leadership immediately
}

// quorum is the number of nodes that make a majority.
func (n *Node) quorum() int { return len(n.peers)/2 + 1 }
