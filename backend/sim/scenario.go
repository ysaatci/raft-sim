package sim

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/ysaatci/raft-sim/backend/raft"
)

// Step is one moment of a scripted scenario: an optional action and the
// narration shown from that moment on.
//
// Targets name nodes by role, because which node leads depends on the run:
//
//	"leader"      the current leader (highest term, if several think they lead)
//	"follower:K"  the K-th running non-leader, counting from 1 in ID order
//	"node:N"      node N
type Step struct {
	At        Time        `json:"at"`
	Narration string      `json:"narration"`
	Kind      ActionKind  `json:"kind,omitempty"`   // empty: narration only
	Target    string      `json:"target,omitempty"` // node acted on; set-link sender
	To        string      `json:"to,omitempty"`     // set-link receiver
	Data      string      `json:"data,omitempty"`   // propose
	Link      *LinkConfig `json:"link,omitempty"`   // set-link, set-network
	Side      []string    `json:"side,omitempty"`   // partition: these nodes vs. everyone else
}

// Scenario is a scripted run that demonstrates one aspect of Raft.
type Scenario struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Config   Config `json:"config"`
	Duration Time   `json:"duration"`
	Steps    []Step `json:"steps"`
}

// Load plays the scenario once from start to finish, turning each step into
// a recorded action with concrete node IDs, then rewinds to time 0. Playing
// it back is then ordinary replay, and acting mid-scenario branches off it
// like any other action.
func Load(sc Scenario) (*Sim, error) {
	s, err := NewSim(sc.Config)
	if err != nil {
		return nil, err
	}
	for i, st := range sc.Steps {
		if st.At < s.Now() {
			return nil, fmt.Errorf("sim: scenario %s: step %d goes back in time", sc.ID, i)
		}
		s.Advance(st.At - s.Now())
		if st.Kind == "" {
			continue
		}
		a, err := resolve(s.Cluster(), st)
		if err == nil {
			err = s.Do(a)
		}
		if err != nil {
			return nil, fmt.Errorf("sim: scenario %s: step %d (%s at %dms): %w", sc.ID, i, st.Kind, st.At, err)
		}
	}
	s.Advance(sc.Duration - s.Now())
	if err := s.Seek(0); err != nil {
		return nil, err
	}
	return s, nil
}

func resolve(c *Cluster, st Step) (Action, error) {
	a := Action{Kind: st.Kind, Data: st.Data, Link: st.Link}
	var err error
	if st.Target != "" {
		if a.Node, err = target(c, st.Target); err != nil {
			return a, err
		}
	}
	if st.To != "" {
		if a.To, err = target(c, st.To); err != nil {
			return a, err
		}
	}
	if st.Side != nil {
		var side []raft.NodeID
		for _, t := range st.Side {
			id, err := target(c, t)
			if err != nil {
				return a, err
			}
			side = append(side, id)
		}
		var rest []raft.NodeID
		for id := raft.NodeID(1); int(id) <= len(c.nodes); id++ {
			if !slices.Contains(side, id) {
				rest = append(rest, id)
			}
		}
		a.Groups = [][]raft.NodeID{side, rest}
	}
	return a, nil
}

func target(c *Cluster, t string) (raft.NodeID, error) {
	kind, arg, _ := strings.Cut(t, ":")
	switch kind {
	case "leader":
		if l := c.Leader(); l != raft.None {
			return l, nil
		}
		return raft.None, fmt.Errorf("no leader at %dms", c.Now())
	case "follower":
		k, err := strconv.Atoi(arg)
		if err != nil || k < 1 {
			return raft.None, fmt.Errorf("bad target %q", t)
		}
		leader := c.Leader()
		for _, sn := range c.running() {
			if sn.id != leader {
				if k--; k == 0 {
					return sn.id, nil
				}
			}
		}
		return raft.None, fmt.Errorf("no %s at %dms", t, c.Now())
	case "node":
		n, err := strconv.Atoi(arg)
		if err != nil {
			return raft.None, fmt.Errorf("bad target %q", t)
		}
		return raft.NodeID(n), nil
	}
	return raft.None, fmt.Errorf("bad target %q", t)
}
