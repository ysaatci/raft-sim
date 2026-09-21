package sim

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/ysaatci/raft-sim/backend/raft"
)

var (
	chaosSeeds = flag.Int("seeds", 200, "number of randomized chaos runs")
	chaosSeed  = flag.Uint64("seed", 0, "replay a single chaos run with this seed")
)

// TestChaos runs many randomized simulations with crashes, restarts, pauses,
// partitions and lossy links, checking every safety property throughout and
// that the cluster recovers once the faults stop.
func TestChaos(t *testing.T) {
	seeds := make([]uint64, 0, *chaosSeeds)
	if *chaosSeed != 0 {
		seeds = append(seeds, *chaosSeed)
	} else {
		n := *chaosSeeds
		if testing.Short() {
			n = min(n, 20)
		}
		for s := range n {
			seeds = append(seeds, uint64(s+1))
		}
	}
	for _, seed := range seeds {
		if err := runChaos(seed); err != nil {
			t.Errorf("seed %d: %v\n  replay: go test ./sim -run TestChaos -seed=%d -v", seed, err, seed)
		}
	}
}

func runChaos(seed uint64) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
	}()
	r := rand.New(rand.NewPCG(seed, 99))
	cfg := DefaultConfig()
	cfg.Seed = seed
	cfg.Size = []int{3, 5, 7}[r.IntN(3)]
	cfg.Network = randomLink(r)
	c, err := NewCluster(cfg)
	if err != nil {
		return err
	}
	ids := others(cfg.Size)

	for step := range 300 {
		c.Advance(Time(10 + r.IntN(90)))
		id := ids[r.IntN(len(ids))]
		switch op := r.IntN(100); {
		case op < 40:
			target := id
			if r.IntN(2) == 0 && c.Leader() != raft.None {
				target = c.Leader()
			}
			if c.NodeState(target) == NodeUp {
				c.Propose(target, fmt.Sprintf("set k%d %d", r.IntN(5), step))
			}
		case op < 50:
			c.Crash(id)
		case op < 60:
			c.Restart(id)
		case op < 65:
			c.Pause(id)
		case op < 72:
			c.Resume(id)
		case op < 80:
			perm := r.Perm(cfg.Size)
			cut := 1 + r.IntN(cfg.Size-1)
			var a, b []raft.NodeID
			for i, p := range perm {
				if i < cut {
					a = append(a, raft.NodeID(p+1))
				} else {
					b = append(b, raft.NodeID(p+1))
				}
			}
			c.Partition(a, b)
		case op < 90:
			c.Heal()
		default:
			c.SetLink(id, ids[r.IntN(len(ids))], randomLink(r))
		}
		if v := c.Violations(); len(v) > 0 {
			return fmt.Errorf("t=%dms %s: %s", v[0].Time, v[0].Invariant, v[0].Detail)
		}
	}

	// Stop the faults and check the cluster recovers and converges.
	for _, id := range ids {
		c.Restart(id)
		c.Resume(id)
		for _, to := range ids {
			c.SetLink(id, to, LinkConfig{LatencyMs: 10})
		}
	}
	c.Advance(3000)
	if v := c.Violations(); len(v) > 0 {
		return fmt.Errorf("t=%dms %s: %s", v[0].Time, v[0].Invariant, v[0].Detail)
	}
	leader := c.Leader()
	if leader == raft.None {
		return fmt.Errorf("no leader after faults stopped")
	}
	// A fresh write proves the cluster makes progress, and replicates everything.
	if err := c.Propose(leader, "set final done"); err != nil {
		return fmt.Errorf("propose after recovery: %v", err)
	}
	c.Advance(1000)
	want := c.KV(leader).Data()
	for _, id := range ids {
		got := c.KV(id).Data()
		if fmt.Sprint(got) != fmt.Sprint(want) || got["final"] != "done" {
			return fmt.Errorf("node %d state %v differs from leader %d state %v", id, got, leader, want)
		}
	}
	return nil
}

func randomLink(r *rand.Rand) LinkConfig {
	return LinkConfig{
		LatencyMs: 5 + r.IntN(40),
		JitterMs:  r.IntN(20),
		DropRate:  []float64{0, 0, 0.05, 0.2, 0.5}[r.IntN(5)],
	}
}
