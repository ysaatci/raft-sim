package raft

import (
	"encoding/json"
	"math/rand/v2"
	"reflect"
	"strings"
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

func TestMessageJSONRoundTrip(t *testing.T) {
	m := Message{Type: MsgAppResp, From: 1, To: 2, Term: 3, Reject: true, ConflictIndex: 4}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"type":"AppendEntriesResp"`) {
		t.Fatalf("type not encoded by name: %s", b)
	}
	var got Message
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, m) {
		t.Fatalf("round trip = %+v, want %+v", got, m)
	}
}

func TestRoleJSON(t *testing.T) {
	b, _ := json.Marshal(Status{Role: Leader})
	if !strings.Contains(string(b), `"role":"leader"`) {
		t.Fatalf("role not encoded by name: %s", b)
	}
	var r Role
	if err := json.Unmarshal([]byte(`"bogus"`), &r); err == nil {
		t.Fatal("unknown role accepted")
	}
}
