# Raft Consensus Simulator — Build Plan

An interactive, visual Raft consensus simulator. The Raft core is written once in Go and runs:

- **Browser mode** — compiled to WebAssembly inside a Web Worker (hosted on GitHub Pages).
- **Cluster mode** — as real processes in Docker Compose, talking over HTTP.

The React + Vite frontend talks to either through one `SimulationClient` interface.

## Ground rules

- Work through the steps **in order**. Tick the box when the step is done.
- **One step ≈ one small commit** (Conventional Commits: `feat:`, `test:`, `chore:`, `ci:`, `docs:`, `refactor:`, `fix:`).
- Every commit builds and its tests pass (`go test ./...`, `npm test`).
- The Raft core (`backend/raft`) is **pure and deterministic**: no goroutines, no timers, no I/O, no global randomness. Time comes from `Tick()`, randomness from an injected seeded RNG.
- Safety invariants are checked in tests from the moment they exist; a failing seed must be replayable.

## Architecture

```
┌──────────────── React + Vite (TS) ────────────────┐
│  ClusterRing · LogGrid · Inspector · Timeline     │
│              SimulationClient interface           │
└──────────┬────────────────────────────┬───────────┘
           │ postMessage                │ WebSocket
   ┌───────▼────────┐          ┌────────▼─────────┐
   │ Web Worker     │          │ Go gateway       │
   │ raft.wasm      │          │ events + chaos   │
   │ sim network +  │          └────────┬─────────┘
   │ virtual clock  │           HTTP between 5 node
   └───────┬────────┘           containers
   ┌───────▼─────────────────────────────▼─────────┐
   │   backend/raft — pure deterministic core      │
   └───────────────────────────────────────────────┘
```

## Repo layout

```
backend/
  raft/         pure core (node, log, election, replication, storage iface)
  sim/          virtual clock, simulated network, cluster driver, invariants, scenarios
  cmd/wasm/     WASM entry point (syscall/js bridge)
  cmd/node/     real node binary (cluster mode)
  cmd/gateway/  WebSocket gateway + fault-injection fan-out
  transport/    HTTP transport + chaos middleware for real nodes
frontend/
  src/client/   SimulationClient, WasmClient, WebSocketClient, MockClient
  src/worker/   Web Worker hosting raft.wasm
  src/store/    Zustand store
  src/components/
  src/scenarios/
docker-compose.yml
Makefile
.github/workflows/
```

## Key decisions

| Topic | Decision |
|---|---|
| Hosting | GitHub Pages, WASM browser mode (no hosted backend) |
| Node-to-node transport (cluster mode) | HTTP/JSON (simpler than gRPC, easy to inspect) |
| Fault injection (cluster mode) | App-level `/chaos` endpoint per node, not Docker network tricks |
| Frontend | React 18, Vite, TypeScript, Tailwind, shadcn/ui, Framer Motion, Zustand |
| Tests | Go `testing` + fuzzing; Vitest + React Testing Library; Playwright smoke |
| PreVote / CheckQuorum | Milestone 8 (after core is proven), enables the "disruptive rejoin" scenario |

---

## Milestone 0 — Repo bootstrap

- [x] 0.1 `docs: add build plan` — plan.md, README stub, .gitignore, LICENSE (MIT)
- [x] 0.2 `chore: init go module` — `backend/go.mod`, empty `raft` package with doc.go
- [x] 0.3 `chore: add Makefile` — targets: `test`, `wasm`, `fe-dev`, `fe-build`, `up`

## Milestone 1 — Raft core (pure, deterministic)

- [x] 1.1 `feat(raft): core types` — `NodeID`, `Term`, `Role`, `Entry`, `Message` (+ `MsgType`), `Config`
- [x] 1.2 `feat(raft): in-memory log` — append, term-at, slice, truncate-from, last index/term; unit tests
- [x] 1.3 `feat(raft): storage interface` — `HardState{Term, VotedFor, Commit}` + log persistence; `MemoryStorage` + tests
- [x] 1.4 `feat(raft): node skeleton` — `NewNode`, `Tick`, `Step`, `Ready()` returning outbound msgs / state / committed entries; follower by default
- [x] 1.5 `feat(raft): randomized election timeout` — seeded RNG injected via Config; follower → candidate on timeout; tests
- [x] 1.6 `feat(raft): RequestVote handling` — term checks, one vote per term, log up-to-date check; tests for each rule
- [x] 1.7 `feat(raft): become leader on majority` — vote counting, step down on higher term; tests
- [x] 1.8 `feat(raft): heartbeats` — leader sends empty AppendEntries on heartbeat interval; followers reset timer; tests
- [x] 1.9 `feat(raft): client proposals` — `Propose(data)` on leader, reject/redirect on follower (leader hint); tests
- [x] 1.10 `feat(raft): AppendEntries consistency check` — prevLogIndex/Term match, conflict truncation, append; tests
- [ ] 1.11 `feat(raft): nextIndex/matchIndex + conflict backtracking` — fast backup via conflict term/index hints; tests
- [ ] 1.12 `feat(raft): commit rule` — majority matchIndex, **only current-term entries**; Figure-8 regression test
- [ ] 1.13 `feat(raft): apply committed entries` — lastApplied, expose in Ready; tests
- [ ] 1.14 `feat(raft): restart from storage` — node rebuilds from HardState + log; tests
- [ ] 1.15 `feat(raft): event stream` — structured events (`BecameLeader`, `VoteGranted`, `EntryCommitted`, ...) for the UI

## Milestone 2 — Simulation harness

- [ ] 2.1 `feat(sim): virtual clock + event queue` — deterministic priority queue by (time, seq)
- [ ] 2.2 `feat(sim): simulated network` — per-link latency, jitter, drop rate, partitions; seeded
- [ ] 2.3 `feat(sim): cluster driver` — N nodes, ticks, message delivery, crash / restart / pause per node
- [ ] 2.4 `feat(sim): KV state machine` — `set k v` commands applied from committed entries
- [ ] 2.5 `feat(sim): invariant checker` — Election Safety, Log Matching, Leader Completeness, State Machine Safety
- [ ] 2.6 `test(sim): election + replication integration tests` — happy path, leader crash, re-election
- [ ] 2.7 `test(sim): randomized chaos tests` — many seeds of random crashes/partitions/drops; failure prints seed; `-seeds` flag
- [ ] 2.8 `test(raft): fuzz Step` — `go test -fuzz` target on message handling (no panics, invariants hold)
- [ ] 2.9 `feat(sim): snapshot/serialize state` — full cluster view for UI (roles, terms, logs, indices, in-flight msgs)
- [ ] 2.10 `feat(sim): history + rewind` — record checkpoints, step back/forward

## Milestone 3 — WASM bridge

- [ ] 3.1 `feat(wasm): js bridge` — `raftSim.create(config)`, `step(n)`, `crash(id)`, `restart(id)`, `setLink(...)`, `propose(...)`, `state()`, `events()`
- [ ] 3.2 `chore: wasm build target` — `GOOS=js GOARCH=wasm`, copy `wasm_exec.js` + `raft.wasm` into `frontend/public`
- [ ] 3.3 `test(wasm): bridge smoke test` — node-based test of the compiled WASM (runs in CI)

## Milestone 4 — Frontend foundation

- [ ] 4.1 `chore(fe): scaffold vite react-ts` — ESLint, Prettier, Vitest config
- [ ] 4.2 `chore(fe): tailwind + shadcn/ui + dark theme tokens`
- [ ] 4.3 `feat(fe): SimulationClient interface + MockClient` — TS types mirroring Go state
- [ ] 4.4 `feat(fe): web worker + WasmClient` — load wasm in worker, message protocol
- [ ] 4.5 `feat(fe): zustand store + sim loop` — play/pause/step/speed driving the client
- [ ] 4.6 `feat(fe): ClusterRing` — SVG ring, role colors, term badge, election-timeout progress ring
- [ ] 4.7 `feat(fe): node actions` — click node: crash / restart / pause; client write to node
- [ ] 4.8 `test(fe): store + ClusterRing tests` — against MockClient

## Milestone 5 — Full interactive UI

- [ ] 5.1 `feat(fe): message packets` — animated in-flight messages by type, dropped packets fade out
- [ ] 5.2 `feat(fe): LogGrid` — nodes × indices, colored by term, commit index marker
- [ ] 5.3 `feat(fe): Inspector panel` — term, votedFor, commit, lastApplied, nextIndex/matchIndex, KV state
- [ ] 5.4 `feat(fe): partitions` — click links to cut; drag-to-partition tool; heal all
- [ ] 5.5 `feat(fe): network controls` — latency / jitter / drop sliders (global + per link)
- [ ] 5.6 `feat(fe): timeline + event feed` — scrubber using sim history, filterable event list
- [ ] 5.7 `feat(fe): cluster settings` — size 3/5/7, seed, timing params, reset
- [ ] 5.8 `feat(fe): invariant badge` — live safety status
- [ ] 5.9 `style(fe): polish` — layout, responsive, keyboard shortcuts, empty/loading states
- [ ] 5.10 `test(fe): component tests` — LogGrid, Inspector, controls

## Milestone 6 — Scenarios + GitHub Pages

- [ ] 6.1 `feat(sim): scenario DSL` — scripted actions at ticks (crash, partition, propose, heal)
- [ ] 6.2 `feat(sim): scenarios` — normal replication, leader crash, split vote, minority partition, divergent logs, packet loss, crash during replication
- [ ] 6.3 `test(sim): scenario outcome tests` — each preset asserts its expected result
- [ ] 6.4 `feat(fe): scenario picker + narration` — guided step-by-step captions
- [ ] 6.5 `feat(fe): shareable URLs` — seed + scenario + config in URL hash
- [ ] 6.6 `ci: test workflow` — go vet, go test -race, short chaos run, wasm build, lint, vitest, build
- [ ] 6.7 `ci: pages deploy workflow` — build wasm + vite (`base: '/raft-sim/'`), `actions/deploy-pages`
- [ ] 6.8 `test(fe): playwright smoke` — load, crash leader, new leader appears
- [ ] 6.9 `docs: README` — screenshots/GIF, live link, how Raft works here, local dev

## Milestone 7 — Cluster mode (Docker Compose)

- [ ] 7.1 `feat(transport): http transport` — POST messages between nodes, JSON codec
- [ ] 7.2 `feat(transport): chaos middleware` — drop/delay per peer, crash, pause via `/chaos`
- [ ] 7.3 `feat(node): node binary` — real-time ticker driving core, file-backed storage, event stream endpoint
- [ ] 7.4 `feat(gateway): websocket gateway` — aggregate node events/state, forward UI commands to `/chaos` + propose
- [ ] 7.5 `feat(fe): WebSocketClient` — mode switch Browser / Live cluster
- [ ] 7.6 `chore: dockerfiles` — multi-stage Go image, nginx frontend image
- [ ] 7.7 `chore: docker-compose` — 5 nodes + gateway + frontend at `localhost:8080`
- [ ] 7.8 `test: transport + gateway tests` — httptest-based, `-race`
- [ ] 7.9 `ci: docker build check`

## Milestone 8 — Stretch

- [ ] 8.1 `feat(raft): PreVote` + "disruptive rejoin" scenario
- [ ] 8.2 `feat(raft): CheckQuorum` — leader steps down without majority contact
- [ ] 8.3 `feat(raft): snapshots + log compaction` + InstallSnapshot
- [ ] 8.4 `feat(raft): single-server membership changes`
- [ ] 8.5 `feat(raft): ReadIndex linearizable reads`
- [ ] 8.6 `ci: nightly long chaos run` (~50k seeds)
