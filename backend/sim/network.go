package sim

import (
	"math/rand/v2"

	"github.com/ysaatci/raft-sim/backend/raft"
)

// LinkConfig describes one direction of the connection between two nodes.
type LinkConfig struct {
	LatencyMs int     `json:"latencyMs"`
	JitterMs  int     `json:"jitterMs"` // delay varies uniformly by ±JitterMs
	DropRate  float64 `json:"dropRate"` // probability in [0, 1] a message is lost
	Cut       bool    `json:"cut"`      // partitioned: nothing gets through
}

type link struct{ from, to raft.NodeID }

// Network decides whether and when each message arrives. Links are
// directional, so asymmetric partitions are possible.
type Network struct {
	rand      *rand.Rand
	defaults  LinkConfig
	overrides map[link]LinkConfig
}

func NewNetwork(r *rand.Rand, defaults LinkConfig) *Network {
	return &Network{rand: r, defaults: defaults, overrides: map[link]LinkConfig{}}
}

// Link returns the configuration of the link from -> to.
func (n *Network) Link(from, to raft.NodeID) LinkConfig {
	if cfg, ok := n.overrides[link{from, to}]; ok {
		return cfg
	}
	return n.defaults
}

// SetLink overrides the configuration of the link from -> to.
func (n *Network) SetLink(from, to raft.NodeID, cfg LinkConfig) {
	n.overrides[link{from, to}] = cfg
}

// SetDefaults changes the configuration of every link without an override.
func (n *Network) SetDefaults(cfg LinkConfig) { n.defaults = cfg }

// Partition cuts every link between nodes in different groups, in both
// directions. Links within a group are left as they are.
func (n *Network) Partition(groups ...[]raft.NodeID) {
	for i, g := range groups {
		for _, h := range groups[i+1:] {
			for _, a := range g {
				for _, b := range h {
					n.setCut(a, b, true)
					n.setCut(b, a, true)
				}
			}
		}
	}
}

// Heal reconnects every cut link, keeping latency and loss settings.
func (n *Network) Heal() {
	n.defaults.Cut = false
	for l, cfg := range n.overrides {
		cfg.Cut = false
		n.overrides[l] = cfg
	}
}

func (n *Network) setCut(from, to raft.NodeID, cut bool) {
	cfg := n.Link(from, to)
	cfg.Cut = cut
	n.SetLink(from, to, cfg)
}

// route decides the fate of a message sent at now: its delivery time, or
// false if it is lost.
func (n *Network) route(from, to raft.NodeID, now Time) (Time, bool) {
	cfg := n.Link(from, to)
	if cfg.Cut {
		return 0, false
	}
	if cfg.DropRate > 0 && n.rand.Float64() < cfg.DropRate {
		return 0, false
	}
	delay := cfg.LatencyMs
	if cfg.JitterMs > 0 {
		delay += n.rand.IntN(2*cfg.JitterMs+1) - cfg.JitterMs
	}
	return now + Time(max(delay, 1)), true
}
