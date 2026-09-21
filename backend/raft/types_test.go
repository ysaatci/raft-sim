package raft

import (
	"math/rand/v2"
	"testing"
)

func TestConfigValidate(t *testing.T) {
	valid := func() Config {
		return Config{ID: 1, Peers: []NodeID{1, 2, 3}, ElectionTick: 10, HeartbeatTick: 1, Rand: rand.New(rand.NewPCG(1, 1)), Storage: NewMemoryStorage()}
	}
	if c := valid(); c.validate() != nil {
		t.Fatalf("valid config rejected: %v", c.validate())
	}
	cases := map[string]func(*Config){
		"no id":                    func(c *Config) { c.ID = None },
		"id not in peers":          func(c *Config) { c.ID = 4 },
		"zero heartbeat":           func(c *Config) { c.HeartbeatTick = 0 },
		"election not > heartbeat": func(c *Config) { c.ElectionTick = 1 },
		"no rand":                  func(c *Config) { c.Rand = nil },
		"no storage":               func(c *Config) { c.Storage = nil },
	}
	for name, mutate := range cases {
		c := valid()
		mutate(&c)
		if c.validate() == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}
