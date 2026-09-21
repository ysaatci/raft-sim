package sim

import (
	"testing"

	"github.com/ysaatci/raft-sim/backend/raft"
)

// FuzzCluster interprets the input as a sequence of (op, arg) byte pairs
// driving a cluster, and fails on any panic or safety violation.
//
//	go test ./sim -run '^$' -fuzz FuzzCluster -fuzztime 60s
func FuzzCluster(f *testing.F) {
	f.Add(uint64(1), []byte{0, 200, 1, 0, 0, 200, 2, 0, 0, 255, 3, 0, 0, 200})
	f.Add(uint64(7), []byte{0, 250, 6, 0b00011, 1, 1, 0, 250, 1, 4, 7, 0, 0, 250})
	f.Fuzz(func(t *testing.T, seed uint64, ops []byte) {
		if len(ops) > 400 {
			ops = ops[:400]
		}
		cfg := DefaultConfig()
		cfg.Seed = seed
		cfg.Size = 3 + int(seed%3)*2
		c, err := NewCluster(cfg)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i+1 < len(ops); i += 2 {
			op, arg := ops[i], ops[i+1]
			id := raft.NodeID(int(arg)%cfg.Size + 1)
			switch op % 9 {
			case 0:
				c.Advance(Time(arg) + 1)
			case 1:
				if l := c.Leader(); l != raft.None {
					c.Propose(l, "set k "+string(rune('a'+arg%26)))
				}
			case 2:
				c.Crash(id)
			case 3:
				c.Restart(id)
			case 4:
				c.Pause(id)
			case 5:
				c.Resume(id)
			case 6: // bit i of arg puts node i+1 on one side of the partition
				var a, b []raft.NodeID
				for n := range cfg.Size {
					if arg&(1<<n) != 0 {
						a = append(a, raft.NodeID(n+1))
					} else {
						b = append(b, raft.NodeID(n+1))
					}
				}
				c.Partition(a, b)
			case 7:
				c.Heal()
			case 8:
				c.SetNetwork(LinkConfig{LatencyMs: int(arg % 50), JitterMs: int(arg % 20), DropRate: float64(arg%4) / 10})
			}
		}
		c.Advance(1000)
		for _, v := range c.Violations() {
			t.Errorf("t=%dms %s: %s", v.Time, v.Invariant, v.Detail)
		}
	})
}
