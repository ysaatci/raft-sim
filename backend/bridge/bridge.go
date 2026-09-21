// Package bridge exposes a simulation through a small JSON-in, JSON-out API.
// It has no dependency on syscall/js, so it is tested like any Go package;
// cmd/wasm only forwards JavaScript calls to it.
package bridge

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ysaatci/raft-sim/backend/sim"
)

var errNoSim = errors.New("bridge: no simulation; call create first")

// API holds the one simulation a page (or Web Worker) works with.
type API struct {
	sim *sim.Sim
}

// DefaultConfig returns the default cluster configuration as JSON.
func DefaultConfig() string {
	b, _ := json.Marshal(sim.DefaultConfig())
	return string(b)
}

// Create starts a new simulation from a JSON config. Fields left out keep
// their default values; an empty string means all defaults.
func (a *API) Create(configJSON string) error {
	cfg := sim.DefaultConfig()
	if configJSON != "" {
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return err
		}
	}
	s, err := sim.NewSim(cfg)
	if err != nil {
		return err
	}
	a.sim = s
	return nil
}

// Advance runs the simulation forward by ms virtual milliseconds.
func (a *API) Advance(ms int) error {
	if a.sim == nil {
		return errNoSim
	}
	if ms < 0 {
		return errors.New("bridge: cannot advance by a negative amount; use seek")
	}
	a.sim.Advance(sim.Time(ms))
	return nil
}

// Seek jumps to virtual time t, forwards or backwards.
func (a *API) Seek(t int) error {
	if a.sim == nil {
		return errNoSim
	}
	if t < 0 {
		return errors.New("bridge: time must not be negative")
	}
	return a.sim.Seek(sim.Time(t))
}

// Do applies a JSON-encoded sim.Action now.
func (a *API) Do(actionJSON string) error {
	if a.sim == nil {
		return errNoSim
	}
	var act sim.Action
	if err := json.Unmarshal([]byte(actionJSON), &act); err != nil {
		return err
	}
	return a.sim.Do(act)
}

// State returns the current sim.State as JSON.
func (a *API) State() (string, error) {
	if a.sim == nil {
		return "", errNoSim
	}
	return marshal(a.sim.Cluster().State())
}

// EventPage is a slice of the event history. After a rewind the history
// can be shorter than a caller's cursor; Total tells the caller to reset.
type EventPage struct {
	Total  int              `json:"total"`
	Events []sim.TimedEvent `json:"events"`
}

// Events returns the events from position since onwards, as JSON.
func (a *API) Events(since int) (string, error) {
	if a.sim == nil {
		return "", errNoSim
	}
	h := a.sim.History()
	since = min(max(since, 0), len(h))
	return marshal(EventPage{Total: len(h), Events: append([]sim.TimedEvent{}, h[since:]...)})
}

// Actions returns the recorded timeline as JSON.
func (a *API) Actions() (string, error) {
	if a.sim == nil {
		return "", errNoSim
	}
	return marshal(append([]sim.Action{}, a.sim.Actions()...))
}

func marshal(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}

// Scenarios returns the built-in guided scenarios as JSON.
func Scenarios() string {
	b, _ := json.Marshal(sim.Scenarios())
	return string(b)
}

// LoadScenario replaces the simulation with a built-in scenario, rewound
// to its start with the whole script recorded.
func (a *API) LoadScenario(id string) error {
	sc, ok := sim.ScenarioByID(id)
	if !ok {
		return fmt.Errorf("bridge: no scenario %q", id)
	}
	s, err := sim.Load(sc)
	if err != nil {
		return err
	}
	a.sim = s
	return nil
}

// Replay replaces the simulation with one rebuilt from a config and a
// recorded timeline, both JSON (as produced by State and Actions).
func (a *API) Replay(configJSON, actionsJSON string) error {
	cfg := sim.DefaultConfig()
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return err
	}
	var actions []sim.Action
	if err := json.Unmarshal([]byte(actionsJSON), &actions); err != nil {
		return err
	}
	s, err := sim.Replay(cfg, actions)
	if err != nil {
		return err
	}
	a.sim = s
	return nil
}
