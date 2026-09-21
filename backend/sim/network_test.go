package sim

import (
	"math/rand/v2"
	"testing"

	"github.com/ysaatci/raft-sim/backend/raft"
)

func newTestNetwork(cfg LinkConfig) *Network {
	return NewNetwork(rand.New(rand.NewPCG(1, 2)), cfg)
}

func TestRouteAppliesLatency(t *testing.T) {
	n := newTestNetwork(LinkConfig{LatencyMs: 20})
	if at, ok := n.route(1, 2, 100); !ok || at != 120 {
		t.Fatalf("route = %d, %v; want 120, true", at, ok)
	}
}

func TestRouteJitterStaysInBounds(t *testing.T) {
	n := newTestNetwork(LinkConfig{LatencyMs: 20, JitterMs: 5})
	seen := map[Time]bool{}
	for range 1000 {
		at, _ := n.route(1, 2, 0)
		if at < 15 || at > 25 {
			t.Fatalf("delay %d outside [15, 25]", at)
		}
		seen[at] = true
	}
	if len(seen) != 11 {
		t.Fatalf("saw %d distinct delays, want all 11", len(seen))
	}
}

func TestRouteNeverDeliversInstantly(t *testing.T) {
	n := newTestNetwork(LinkConfig{LatencyMs: 0})
	if at, _ := n.route(1, 2, 50); at != 51 {
		t.Fatalf("delivery at %d, want 51", at)
	}
}

func TestRouteDropRate(t *testing.T) {
	n := newTestNetwork(LinkConfig{LatencyMs: 1, DropRate: 0.3})
	dropped := 0
	for range 10000 {
		if _, ok := n.route(1, 2, 0); !ok {
			dropped++
		}
	}
	if dropped < 2700 || dropped > 3300 {
		t.Fatalf("dropped %d/10000, want about 3000", dropped)
	}
}

func TestPartitionCutsOnlyCrossGroupLinks(t *testing.T) {
	n := newTestNetwork(LinkConfig{LatencyMs: 1})
	n.Partition([]raft.NodeID{1, 2}, []raft.NodeID{3})
	for _, l := range []link{{1, 3}, {3, 1}, {2, 3}, {3, 2}} {
		if _, ok := n.route(l.from, l.to, 0); ok {
			t.Errorf("%d -> %d delivered across partition", l.from, l.to)
		}
	}
	if _, ok := n.route(1, 2, 0); !ok {
		t.Error("1 -> 2 cut within a group")
	}
}

func TestHealKeepsLinkSettings(t *testing.T) {
	n := newTestNetwork(LinkConfig{LatencyMs: 1})
	n.SetLink(1, 2, LinkConfig{LatencyMs: 50})
	n.Partition([]raft.NodeID{1}, []raft.NodeID{2})
	n.Heal()
	if at, ok := n.route(1, 2, 0); !ok || at != 50 {
		t.Fatalf("after heal route = %d, %v; want 50, true", at, ok)
	}
}

func TestAsymmetricLink(t *testing.T) {
	n := newTestNetwork(LinkConfig{LatencyMs: 1})
	n.SetLink(1, 2, LinkConfig{Cut: true})
	if _, ok := n.route(1, 2, 0); ok {
		t.Error("1 -> 2 should be cut")
	}
	if _, ok := n.route(2, 1, 0); !ok {
		t.Error("2 -> 1 should still work")
	}
}

func TestNetworkIsDeterministic(t *testing.T) {
	cfg := LinkConfig{LatencyMs: 10, JitterMs: 10, DropRate: 0.5}
	a, b := newTestNetwork(cfg), newTestNetwork(cfg)
	for range 100 {
		at1, ok1 := a.route(1, 2, 0)
		at2, ok2 := b.route(1, 2, 0)
		if at1 != at2 || ok1 != ok2 {
			t.Fatal("same seed produced different routes")
		}
	}
}
