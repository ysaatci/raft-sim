import type { Frame, SimulationClient } from './client'
import type { Action, Config, NodeView, Scenario, SimEvent, SimState } from './types'

const nodeEvents = {
  crash: 'node-crashed',
  restart: 'node-restarted',
  pause: 'node-paused',
  resume: 'node-resumed',
} as const satisfies Record<string, SimEvent['type']>

const defaultConfig: Config = {
  size: 5,
  seed: 1,
  tickMs: 10,
  electionTick: 15,
  heartbeatTick: 5,
  network: { latencyMs: 20, jitterMs: 5, dropRate: 0, cut: false },
}

export const MOCK_SCENARIOS: Scenario[] = [
  {
    id: 'demo',
    title: 'Demo',
    summary: 'A scripted demo.',
    config: { ...defaultConfig, size: 3, seed: 9 },
    duration: 1000,
    steps: [
      { at: 0, narration: 'First.' },
      { at: 300, narration: 'Second.', kind: 'propose' },
      { at: 500, narration: '' },
      { at: 800, narration: 'Third.' },
    ],
  },
  {
    id: 'other',
    title: 'Other',
    summary: 'Another one.',
    config: defaultConfig,
    duration: 500,
    steps: [{ at: 0, narration: 'Hi.' }],
  },
]

function node(id: number): NodeView {
  return {
    id,
    role: 'follower',
    term: 0,
    votedFor: 0,
    leader: 0,
    commit: 0,
    applied: 0,
    lastIndex: 0,
    lastTerm: 0,
    electionElapsed: 0,
    electionTimeout: 20,
    state: 'up',
    log: [],
    kv: {},
    inbox: 0,
  }
}

/**
 * A scripted, synchronous stand-in for the simulator. Node 1 becomes leader
 * at 300ms; actions change node states directly. Every call is recorded in
 * `calls` so tests can assert what the UI asked for.
 */
export class MockClient implements SimulationClient {
  calls: string[] = []
  state!: SimState
  private pending: SimEvent[] = []

  async create(config?: Partial<Config>): Promise<Frame> {
    const cfg = { ...defaultConfig, ...config }
    this.calls.push(`create ${JSON.stringify(config ?? {})}`)
    this.state = {
      time: 0,
      config: cfg,
      nodes: Array.from({ length: cfg.size }, (_, i) => node(i + 1)),
      flights: [],
      network: { ...cfg.network },
      links: [],
      violations: [],
    }
    for (let from = 1; from <= cfg.size; from++) {
      for (let to = 1; to <= cfg.size; to++) {
        if (from !== to) this.state.links.push({ from, to, ...cfg.network })
      }
    }
    return this.frame(true)
  }

  async scenarios(): Promise<Scenario[]> {
    return structuredClone(MOCK_SCENARIOS)
  }

  async loadScenario(id: string): Promise<Frame> {
    const sc = MOCK_SCENARIOS.find((s) => s.id === id)
    if (!sc) throw new Error(`bridge: no scenario "${id}"`)
    const frame = await this.create(sc.config)
    this.calls[this.calls.length - 1] = `scenario ${id}`
    return frame
  }

  async advance(ms: number): Promise<Frame> {
    this.calls.push(`advance ${ms}`)
    const before = this.state.time
    this.state.time += ms
    const first = this.state.nodes[0]
    if (before < 300 && this.state.time >= 300 && first.state === 'up') {
      for (const n of this.state.nodes) {
        n.term = 1
        n.leader = 1
      }
      first.role = 'leader'
      this.pending.push({ time: 300, type: 'became-leader', node: 1, term: 1 })
    }
    return this.frame(false)
  }

  async seek(time: number): Promise<Frame> {
    this.calls.push(`seek ${time}`)
    this.state.time = time
    return this.frame(true)
  }

  async do(action: Action): Promise<Frame> {
    this.calls.push(`do ${JSON.stringify(action)}`)
    if (action.kind === 'propose' && this.state.nodes[action.node - 1].role !== 'leader') {
      throw new Error('raft: not leader, try node 1')
    }
    if (
      action.kind === 'crash' ||
      action.kind === 'restart' ||
      action.kind === 'pause' ||
      action.kind === 'resume'
    ) {
      const n = this.state.nodes[action.node - 1]
      n.state = action.kind === 'crash' ? 'crashed' : action.kind === 'pause' ? 'paused' : 'up'
      if (n.state !== 'up') n.role = 'follower'
      this.pending.push({ time: this.state.time, type: nodeEvents[action.kind], node: n.id, term: n.term })
    }
    const links = this.state.links
    if (action.kind === 'set-link') {
      const l = links.find((l) => l.from === action.node && l.to === action.to)!
      Object.assign(l, action.link)
    }
    if (action.kind === 'partition') {
      const side = (id: number) => action.groups.findIndex((g) => g.includes(id))
      for (const l of links) if (side(l.from) !== side(l.to)) l.cut = true
    }
    if (action.kind === 'heal') for (const l of links) l.cut = false
    if (action.kind === 'set-network') {
      this.state.network = { ...action.link, cut: false }
      for (const l of links) Object.assign(l, { ...action.link, cut: l.cut })
    }
    return this.frame(false)
  }

  dispose() {
    this.calls.push('dispose')
  }

  private frame(reset: boolean): Frame {
    const events = this.pending
    this.pending = []
    return { state: structuredClone(this.state), events, reset }
  }
}
