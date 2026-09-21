package bridge

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/ysaatci/raft-sim/backend/raft"
	"github.com/ysaatci/raft-sim/backend/sim"
)

func mustState(t *testing.T, a *API) sim.State {
	t.Helper()
	js, err := a.State()
	if err != nil {
		t.Fatal(err)
	}
	var s sim.State
	if err := json.Unmarshal([]byte(js), &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func leaderOf(s sim.State) raft.NodeID {
	for _, n := range s.Nodes {
		if n.Role == raft.Leader {
			return n.ID
		}
	}
	return raft.None
}

func TestCallsBeforeCreateFail(t *testing.T) {
	var a API
	if a.Advance(10) == nil || a.Seek(0) == nil || a.Do(`{"kind":"heal"}`) == nil {
		t.Fatal("mutations before create should fail")
	}
	if _, err := a.State(); err == nil {
		t.Fatal("State before create should fail")
	}
}

func TestCreateMergesPartialConfig(t *testing.T) {
	var a API
	if err := a.Create(`{"size": 3, "seed": 9}`); err != nil {
		t.Fatal(err)
	}
	s := mustState(t, &a)
	if s.Config.Size != 3 || s.Config.Seed != 9 || s.Config.TickMs != sim.DefaultConfig().TickMs {
		t.Fatalf("config = %+v", s.Config)
	}
	if err := a.Create(`{"size": 0}`); err == nil {
		t.Fatal("invalid config accepted")
	}
	if err := a.Create(`{bad json`); err == nil {
		t.Fatal("malformed config accepted")
	}
}

func TestDefaultConfigJSON(t *testing.T) {
	var cfg sim.Config
	if err := json.Unmarshal([]byte(DefaultConfig()), &cfg); err != nil || cfg != sim.DefaultConfig() {
		t.Fatalf("DefaultConfig() = %s, %v", DefaultConfig(), err)
	}
}

func TestDriveSimulation(t *testing.T) {
	var a API
	a.Create("")
	if err := a.Advance(1000); err != nil {
		t.Fatal(err)
	}
	leader := leaderOf(mustState(t, &a))
	if leader == raft.None {
		t.Fatal("no leader after 1s")
	}
	if err := a.Do(`{"kind":"propose","node":` + fmt.Sprint(leader) + `,"data":"set x 1"}`); err != nil {
		t.Fatal(err)
	}
	if err := a.Do(`{"kind":"propose","node":` + fmt.Sprint(leader%5+1) + `,"data":"set x 2"}`); err == nil {
		t.Fatal("propose to follower succeeded")
	}
	a.Advance(500)
	if kv := mustState(t, &a).Nodes[0].KV; kv["x"] != "1" {
		t.Fatalf("node 1 kv = %v", kv)
	}
	if err := a.Advance(-1); err == nil {
		t.Fatal("negative advance accepted")
	}
}

func TestEventsPaging(t *testing.T) {
	var a API
	a.Create("")
	a.Advance(1000)
	var page EventPage
	js, _ := a.Events(0)
	json.Unmarshal([]byte(js), &page)
	if page.Total == 0 || len(page.Events) != page.Total {
		t.Fatalf("first page: total %d, %d events", page.Total, len(page.Events))
	}
	total := page.Total

	js, _ = a.Events(total + 50) // cursor past the end is clamped
	json.Unmarshal([]byte(js), &page)
	if len(page.Events) != 0 || page.Total != total {
		t.Fatalf("clamped page = %+v", page)
	}

	a.Seek(10) // rewind: history shrinks below the caller's cursor
	js, _ = a.Events(total)
	json.Unmarshal([]byte(js), &page)
	if page.Total >= total {
		t.Fatalf("total after rewind = %d, want < %d", page.Total, total)
	}
}

func TestActionsJSON(t *testing.T) {
	var a API
	a.Create("")
	a.Advance(100)
	a.Do(`{"kind":"crash","node":2}`)
	js, _ := a.Actions()
	if !strings.Contains(js, `"kind":"crash"`) || !strings.Contains(js, `"at":100`) {
		t.Fatalf("actions = %s", js)
	}
}

func TestScenariosAndLoadScenario(t *testing.T) {
	var list []sim.Scenario
	if err := json.Unmarshal([]byte(Scenarios()), &list); err != nil || len(list) == 0 {
		t.Fatalf("Scenarios() = %d scenarios, %v", len(list), err)
	}
	var a API
	if err := a.LoadScenario("nope"); err == nil {
		t.Fatal("unknown scenario loaded")
	}
	if err := a.LoadScenario(list[0].ID); err != nil {
		t.Fatal(err)
	}
	if s := mustState(t, &a); s.Time != 0 || s.Config != list[0].Config {
		t.Fatalf("loaded state: time %d, config %+v", s.Time, s.Config)
	}
	acts, _ := a.Actions()
	if !strings.Contains(acts, `"kind":"propose"`) {
		t.Fatalf("scenario script not recorded: %s", acts)
	}
}
