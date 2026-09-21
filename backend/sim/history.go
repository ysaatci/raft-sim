package sim

import (
	"fmt"

	"github.com/ysaatci/raft-sim/backend/raft"
)

// ActionKind names something a user (or scenario) does to the cluster.
type ActionKind string

const (
	ActCrash      ActionKind = "crash"
	ActRestart    ActionKind = "restart"
	ActPause      ActionKind = "pause"
	ActResume     ActionKind = "resume"
	ActPropose    ActionKind = "propose"
	ActSetLink    ActionKind = "set-link"
	ActSetNetwork ActionKind = "set-network"
	ActPartition  ActionKind = "partition"
	ActHeal       ActionKind = "heal"
)

// Action is one recorded intervention, applied at virtual time At.
type Action struct {
	At     Time            `json:"at"`
	Kind   ActionKind      `json:"kind"`
	Node   raft.NodeID     `json:"node,omitempty"`   // target; for set-link the sender
	To     raft.NodeID     `json:"to,omitempty"`     // set-link receiver
	Data   string          `json:"data,omitempty"`   // propose
	Link   *LinkConfig     `json:"link,omitempty"`   // set-link, set-network
	Groups [][]raft.NodeID `json:"groups,omitempty"` // partition
}

func (a Action) apply(c *Cluster) error {
	switch a.Kind {
	case ActCrash:
		return c.Crash(a.Node)
	case ActRestart:
		return c.Restart(a.Node)
	case ActPause:
		return c.Pause(a.Node)
	case ActResume:
		return c.Resume(a.Node)
	case ActPropose:
		return c.Propose(a.Node, a.Data)
	case ActSetLink:
		if a.Link == nil {
			return fmt.Errorf("sim: %s needs a link", a.Kind)
		}
		if _, err := c.lookup(a.Node); err != nil {
			return err
		}
		if _, err := c.lookup(a.To); err != nil {
			return err
		}
		c.SetLink(a.Node, a.To, *a.Link)
	case ActSetNetwork:
		if a.Link == nil {
			return fmt.Errorf("sim: %s needs a link", a.Kind)
		}
		c.SetNetwork(*a.Link)
	case ActPartition:
		for _, g := range a.Groups {
			for _, id := range g {
				if _, err := c.lookup(id); err != nil {
					return err
				}
			}
		}
		c.Partition(a.Groups...)
	case ActHeal:
		c.Heal()
	default:
		return fmt.Errorf("sim: unknown action %q", a.Kind)
	}
	return nil
}

// Sim is a cluster with a recorded timeline. Because runs are
// deterministic, any past moment can be reconstructed exactly by
// rebuilding the cluster from its seed and replaying the actions.
type Sim struct {
	cfg     Config
	c       *Cluster
	actions []Action     // the timeline, in time order
	applied int          // actions[:applied] have been applied to c
	history []TimedEvent // every event up to the current time
}

func NewSim(cfg Config) (*Sim, error) {
	c, err := NewCluster(cfg)
	if err != nil {
		return nil, err
	}
	return &Sim{cfg: cfg, c: c}, nil
}

// Cluster returns the current cluster, for reading state. Mutate it only
// through the Sim, or the change will not survive a rewind.
func (s *Sim) Cluster() *Cluster { return s.c }

// Now returns the current virtual time.
func (s *Sim) Now() Time { return s.c.Now() }

// Actions returns the recorded timeline.
func (s *Sim) Actions() []Action { return s.actions }

// History returns every event up to the current time.
func (s *Sim) History() []TimedEvent { return s.history }

// Do applies an action now and records it. After a rewind, this starts a
// new branch: recorded actions later than now are discarded.
func (s *Sim) Do(a Action) error {
	a.At = s.c.Now()
	if err := a.apply(s.c); err != nil {
		s.drain()
		return err
	}
	s.actions = append(s.actions[:s.applied], a)
	s.applied++
	s.drain()
	return nil
}

// Advance moves forward by ms, replaying recorded actions on the way.
func (s *Sim) Advance(ms Time) { s.replayTo(s.c.Now() + ms) }

// Seek moves to virtual time t, forwards or backwards.
func (s *Sim) Seek(t Time) error {
	if t < s.c.Now() {
		c, err := NewCluster(s.cfg)
		if err != nil {
			return err
		}
		s.c, s.applied, s.history = c, 0, nil
	}
	s.replayTo(t)
	return nil
}

// replayTo advances to t, applying recorded actions stamped before t. Actions
// stamped exactly t stay pending: the state "at" t is the state before them,
// matching what the user saw when they acted.
func (s *Sim) replayTo(t Time) {
	for s.applied < len(s.actions) && s.actions[s.applied].At < t {
		a := s.actions[s.applied]
		s.c.Advance(a.At - s.c.Now())
		a.apply(s.c) // errors are deterministic and were already reported
		s.applied++
	}
	s.c.Advance(t - s.c.Now())
	s.drain()
}

func (s *Sim) drain() { s.history = append(s.history, s.c.Events()...) }
