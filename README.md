# raft-sim

An interactive, visual simulator of the [Raft consensus algorithm](https://raft.github.io/).
Crash nodes, cut network links, add latency and packet loss, and watch leader election
and log replication respond in real time.

- **Browser mode** — the Go Raft core compiled to WebAssembly, hosted on GitHub Pages.
- **Cluster mode** — the same core running as 5 real nodes via Docker Compose.

> 🚧 Work in progress — see [plan.md](plan.md) for the step-by-step build plan.
