package sim

import (
	"maps"
	"strings"
)

// KV is the replicated state machine: a string map driven by commands of
// the form "set <key> <value>". Other entries (such as a leader's no-op)
// are ignored.
type KV struct {
	data map[string]string
}

func NewKV() *KV { return &KV{data: map[string]string{}} }

// Apply executes one committed command.
func (kv *KV) Apply(cmd string) {
	parts := strings.SplitN(cmd, " ", 3)
	if len(parts) == 3 && parts[0] == "set" {
		kv.data[parts[1]] = parts[2]
	}
}

// Get returns the value stored under key.
func (kv *KV) Get(key string) (string, bool) {
	v, ok := kv.data[key]
	return v, ok
}

// Data returns a copy of the whole map.
func (kv *KV) Data() map[string]string { return maps.Clone(kv.data) }
