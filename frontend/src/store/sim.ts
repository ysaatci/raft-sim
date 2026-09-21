import { createStore, useStore } from 'zustand'
import { linkStatus } from '@/lib/cluster'
import type { Frame, SimulationClient } from '@/client/client'
import type { Action, Config, NodeID, SimEvent, SimState } from '@/client/types'

export const SPEEDS = [0.02, 0.05, 0.1, 0.25, 1] as const
export const DEFAULT_SPEED = 0.05
const MAX_EVENTS = 2000
// Longest virtual step per animation frame, so a slow frame or a stalled
// request never makes the simulation jump ahead abruptly.
const MAX_STEP_MS = 50

export interface SimStore {
  client: SimulationClient | null
  state: SimState | null
  events: SimEvent[]
  playing: boolean
  /** Virtual milliseconds per real millisecond. */
  speed: number
  /** Last rejected action, e.g. proposing to a follower. */
  error: string | null
  /** Node shown in the side panel. */
  selected: NodeID | null
  /** Nodes picked for one side of a partition being drawn, or null. */
  partitionDraft: NodeID[] | null
  /** Furthest virtual time on the current timeline; the end of the scrubber. */
  horizon: number

  init(client: SimulationClient, config?: Partial<Config>): Promise<void>
  /** Starts over on the same client, with a new or the current config. */
  reset(config?: Partial<Config>): Promise<void>
  play(): void
  pause(): void
  setSpeed(speed: number): void
  /** Called every animation frame with the real time elapsed. */
  tick(realMs: number): Promise<void>
  step(ms: number): Promise<void>
  seek(time: number): Promise<void>
  /** Applies an action; resolves false if the simulator rejected it. */
  act(action: Action): Promise<boolean>
  clearError(): void
  select(id: NodeID | null): void
  /** Cuts the link between a and b in both directions, or restores it if any direction is cut. */
  toggleLink(a: NodeID, b: NodeID): Promise<void>
  startPartition(): void
  togglePartitionNode(id: NodeID): void
  /** Cuts every link between the picked nodes and the rest. */
  applyPartition(): Promise<void>
  cancelPartition(): void
}

export function createSimStore() {
  let queue: Promise<void> = Promise.resolve()
  let pending = 0 // queued or running requests; animation frames skip rather than pile up
  let carry = 0 // virtual time owed but not yet advanced

  return createStore<SimStore>()((set, get) => {
    const apply = (f: Frame) =>
      set((s) => ({
        state: f.state,
        events: f.reset ? f.events.slice(-MAX_EVENTS) : [...s.events, ...f.events].slice(-MAX_EVENTS),
        horizon: Math.max(s.horizon, f.state.time),
      }))

    // Requests run one at a time, in order: each waits for the previous one.
    const run = (req: (c: SimulationClient) => Promise<Frame>): Promise<void> => {
      const client = get().client
      if (!client) return Promise.resolve()
      pending++
      const p = queue.then(async () => {
        try {
          apply(await req(client))
        } finally {
          pending--
        }
      })
      queue = p.catch(() => {}) // one failure must not block later requests
      return p
    }

    return {
      client: null,
      state: null,
      events: [],
      playing: false,
      speed: DEFAULT_SPEED,
      selected: null,
      error: null,
      partitionDraft: null,
      horizon: 0,

      async init(client, config) {
        get().client?.dispose()
        queue = Promise.resolve()
        pending = 0
        carry = 0
        set({
          client,
          state: null,
          events: [],
          error: null,
          selected: null,
          partitionDraft: null,
          horizon: 0,
        })
        apply(await client.create(config))
      },
      async reset(config) {
        const cfg = config ?? get().state?.config
        carry = 0
        await run((c) => {
          // Clear inside the queue, after any in-flight request has landed.
          set({ events: [], selected: null, partitionDraft: null, horizon: 0, error: null })
          return c.create(cfg)
        })
      },
      play: () => set({ playing: true }),
      pause: () => set({ playing: false }),
      setSpeed: (speed) => set({ speed }),

      async tick(realMs) {
        if (!get().playing) return
        carry = Math.min(carry + realMs * get().speed, MAX_STEP_MS)
        const ms = Math.floor(carry)
        if (ms < 1) return
        if (pending > 0) return
        carry -= ms
        await run((c) => c.advance(ms))
      },
      async step(ms) {
        await run((c) => c.advance(ms))
      },
      async seek(time) {
        carry = 0
        await run((c) => c.seek(Math.max(0, Math.round(time))))
      },
      async act(action) {
        try {
          let branched = false
          await run((c) => {
            const s = get()
            branched = !!s.state && s.state.time < s.horizon
            return c.do(action)
          })
          // Acting in the past starts a new branch: the recorded future is gone.
          set(branched ? { error: null, horizon: get().state!.time } : { error: null })
          return true
        } catch (e) {
          set({ error: e instanceof Error ? e.message : String(e) })
          return false
        }
      },
      clearError: () => set({ error: null }),
      select: (selected) => set({ selected }),

      async toggleLink(a, b) {
        const links = get().state?.links ?? []
        const cut = linkStatus(links, a, b) === 'up'
        for (const [from, to] of [
          [a, b],
          [b, a],
        ]) {
          const l = links.find((l) => l.from === from && l.to === to)
          if (!l) continue
          const { latencyMs, jitterMs, dropRate } = l
          await get().act({ kind: 'set-link', node: from, to, link: { latencyMs, jitterMs, dropRate, cut } })
        }
      },
      startPartition: () => set({ partitionDraft: [], selected: null }),
      togglePartitionNode: (id) =>
        set((s) => ({
          partitionDraft: s.partitionDraft?.includes(id)
            ? s.partitionDraft.filter((n) => n !== id)
            : [...(s.partitionDraft ?? []), id],
        })),
      async applyPartition() {
        const { partitionDraft: side, state } = get()
        if (!side || !state) return
        const rest = state.nodes.map((n) => n.id).filter((id) => !side.includes(id))
        set({ partitionDraft: null })
        if (side.length && rest.length) await get().act({ kind: 'partition', groups: [side, rest] })
      },
      cancelPartition: () => set({ partitionDraft: null }),
    }
  })
}

export const simStore = createSimStore()

export function useSim<T>(selector: (s: SimStore) => T): T {
  return useStore(simStore, selector)
}
