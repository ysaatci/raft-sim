package sim

// The built-in guided scenarios. Each fixes its seed, so every viewer sees
// the same run; scenario_outcomes_test.go checks that each still shows what
// its narration claims.

func scenarioConfig(size int, seed uint64) Config {
	cfg := DefaultConfig()
	cfg.Size, cfg.Seed = size, seed
	return cfg
}

var lossy = LinkConfig{LatencyMs: 20, JitterMs: 15, DropRate: 0.3}
var healthy = DefaultConfig().Network

// Scenarios lists the built-in scenarios, in the order they are presented.
func Scenarios() []Scenario {
	return []Scenario{
		{
			ID:       "normal-replication",
			Title:    "Normal replication",
			Summary:  "A leader is elected, then replicates and commits client writes.",
			Config:   scenarioConfig(5, 1),
			Duration: 2200,
			Steps: []Step{
				{At: 0, Narration: "Every node starts as a follower. Each has a randomized election timeout (the arc around it). The first to run out becomes a candidate and asks the others for votes."},
				{At: 400, Narration: "A candidate with votes from a majority becomes leader. It appends an empty no-op entry for its term and sends heartbeats (empty AppendEntries) so nobody else starts an election."},
				{At: 700, Narration: "A client sends a write to the leader. The leader appends it to its log and sends it to every follower in AppendEntries.", Kind: ActPropose, Target: "leader", Data: "set color blue"},
				{At: 900, Narration: "Once a majority has stored the entry, the leader commits it (solid cell) and applies it. Followers learn the new commit index from the next AppendEntries and apply it too."},
				{At: 1300, Narration: "Every write follows the same path: leader log, majority replication, commit, apply. All state machines end up identical.", Kind: ActPropose, Target: "leader", Data: "set size large"},
			},
		},
		{
			ID:       "leader-crash",
			Title:    "Leader crash",
			Summary:  "The leader dies; the followers elect a new one, and the old leader rejoins as a follower.",
			Config:   scenarioConfig(5, 2),
			Duration: 3800,
			Steps: []Step{
				{At: 0, Narration: "A cluster elects its first leader."},
				{At: 600, Narration: "A write is committed on every node.", Kind: ActPropose, Target: "leader", Data: "set x 1"},
				{At: 1000, Narration: "The leader crashes. Its term, vote and log are on disk, but it sends nothing, so the heartbeats stop.", Kind: ActCrash, Target: "leader"},
				{At: 1200, Narration: "Without heartbeats, the followers' election timers run out. The first to expire becomes a candidate for the next term."},
				{At: 1700, Narration: "A candidate wins a majority and leads the new term. Only a node whose log is at least as up to date as a majority's can win, so it has every committed entry."},
				{At: 2200, Narration: "The new leader serves writes. The entry committed before the crash is still there.", Kind: ActPropose, Target: "leader", Data: "set x 2"},
				{At: 2600, Narration: "The old leader restarts. It reloads its state from disk, hears the higher term, and follows the new leader, which sends it every entry it missed.", Kind: ActRestart, Target: "crashed:1"},
			},
		},
		{
			ID:       "split-vote",
			Title:    "Split vote",
			Summary:  "Two candidates split the votes and nobody wins, until randomized timeouts break the tie.",
			Config:   scenarioConfig(4, 22),
			Duration: 1200,
			Steps: []Step{
				{At: 0, Narration: "Four nodes, so winning an election takes three votes. Watch the election timers."},
				{At: 140, Narration: "N3 and N2 time out 2ms apart and both become candidates for term 1. Each votes for itself."},
				{At: 170, Narration: "N1 votes for N2 and N4 votes for N3. Each candidate has two votes: a split vote. Nobody can win term 1."},
				{At: 320, Narration: "Each candidate waits a new random timeout. N2 expires first and starts term 2, a few milliseconds ahead of N3."},
				{At: 370, Narration: "N2 collects three votes and leads term 2; N3 sees the higher-term leader and steps down. Randomized timeouts make repeated splits unlikely."},
			},
		},
		{
			ID:       "minority-partition",
			Title:    "Network partition",
			Summary:  "An isolated leader keeps accepting writes it can never commit; the majority moves on without it.",
			Config:   scenarioConfig(5, 3),
			Duration: 3800,
			Steps: []Step{
				{At: 0, Narration: "A leader is elected and commits a first write."},
				{At: 600, Narration: "", Kind: ActPropose, Target: "leader", Data: "set x original"},
				{At: 1000, Narration: "The network splits: the leader and one follower on one side, three nodes on the other.", Kind: ActPartition, Side: []string{"leader", "follower:1"}},
				{At: 1100, Narration: "The old leader doesn't know it is cut off and still accepts a write. It can only reach one follower, never a majority, so the entry stays uncommitted (outlined).", Kind: ActPropose, Target: "leader", Data: "set x stale"},
				{At: 1400, Narration: "On the majority side the heartbeats stopped, so followers time out and run for leader. Here the first rounds split the vote; after a few terms one candidate wins. Two nodes now think they lead, in different terms."},
				{At: 2000, Narration: "The new leader commits writes normally, because it can reach a majority.", Kind: ActPropose, Target: "leader", Data: "set x fresh"},
				{At: 2600, Narration: "The partition heals. The old leader sees the higher term and steps down, and its uncommitted entry is overwritten by the new leader's log. No committed data was lost.", Kind: ActHeal},
			},
		},
		{
			ID:       "divergent-logs",
			Title:    "Divergent logs",
			Summary:  "A follower holds entries the rest of the cluster never saw; the next leader repairs its log.",
			Config:   scenarioConfig(5, 4),
			Duration: 3000,
			Steps: []Step{
				{At: 0, Narration: "A leader is elected and commits a write on every node."},
				{At: 600, Narration: "", Kind: ActPropose, Target: "leader", Data: "set a 1"},
				{At: 900, Narration: "The leader and one follower are cut off from the others.", Kind: ActPartition, Side: []string{"leader", "follower:1"}},
				{At: 950, Narration: "The leader accepts two writes. They reach only its one connected follower.", Kind: ActPropose, Target: "leader", Data: "set b 2"},
				{At: 980, Kind: ActPropose, Target: "leader", Data: "set c 3"},
				{At: 1000, Narration: "Then the leader crashes. Its follower now holds two entries that no other node has, and that were never committed.", Kind: ActCrash, Target: "leader"},
				{At: 1300, Narration: "The majority elects a new leader, which appends its own no-op for the new term at the same index."},
				{At: 1500, Narration: "The network heals. The new leader's AppendEntries fail the consistency check on the diverged follower, so it backs up until the logs agree, then overwrites the conflicting entries.", Kind: ActHeal},
				{At: 1900, Narration: "All running logs are identical again. Entries only a minority stored can be discarded; committed ones never are.", Kind: ActPropose, Target: "leader", Data: "set d 4"},
			},
		},
		{
			ID:       "packet-loss",
			Title:    "Packet loss",
			Summary:  "30% of messages are lost; retries keep the cluster consistent, if slower.",
			Config:   scenarioConfig(5, 5),
			Duration: 4000,
			Steps: []Step{
				{At: 0, Narration: "A healthy cluster elects a leader."},
				{At: 600, Narration: "The network degrades: 30% of messages are dropped and delays vary more. Dropped messages fade out halfway.", Kind: ActSetNetwork, Link: &lossy},
				{At: 900, Narration: "Writes still succeed. A lost AppendEntries is simply resent with the next heartbeat, and a lost response just delays the commit.", Kind: ActPropose, Target: "leader", Data: "set k 1"},
				{At: 1500, Narration: "If a follower misses heartbeats for a whole election timeout, it starts an election. Disruptions cost time, never consistency.", Kind: ActPropose, Target: "leader", Data: "set k 2"},
				{At: 2100, Kind: ActPropose, Target: "leader", Data: "set k 3"},
				{At: 2800, Narration: "The network recovers and every node converges on the same log.", Kind: ActSetNetwork, Link: &healthy},
			},
		},
		{
			ID:       "crash-during-replication",
			Title:    "Crash during replication",
			Summary:  "The leader dies with a write in flight. The write survives because a majority stored it.",
			Config:   scenarioConfig(5, 6),
			Duration: 2800,
			Steps: []Step{
				{At: 0, Narration: "A leader is elected."},
				{At: 600, Narration: "A client sends a write. The leader appends it and sends AppendEntries.", Kind: ActPropose, Target: "leader", Data: "set x 1"},
				{At: 605, Narration: "5ms later, before any follower has replied, the leader crashes. It never learns whether the write was committed.", Kind: ActCrash, Target: "leader"},
				{At: 640, Narration: "The AppendEntries were already on the wire: the followers store the entry, but their acknowledgements go nowhere."},
				{At: 1000, Narration: "A follower wins the next election. It has the entry, and commits it together with its own no-op. A write is safe once a majority stores it, even if the leader that proposed it never finds out."},
				{At: 1600, Narration: "The old leader restarts and catches up. The client's write is on every node.", Kind: ActRestart, Target: "crashed:1"},
			},
		},
	}
}

// ScenarioByID returns the built-in scenario with the given ID.
func ScenarioByID(id string) (Scenario, bool) {
	for _, s := range Scenarios() {
		if s.ID == id {
			return s, true
		}
	}
	return Scenario{}, false
}
