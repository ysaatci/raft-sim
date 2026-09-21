import { createStore, useStore } from 'zustand'
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

  init(client: SimulationClient, config?: Partial<Config>): Promise<void>
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

      async init(client, config) {
        get().client?.dispose()
        queue = Promise.resolve()
        pending = 0
        carry = 0
        set({ client, state: null, events: [], error: null, selected: null })
        apply(await client.create(config))
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
          await run((c) => c.do(action))
          set({ error: null })
          return true
        } catch (e) {
          set({ error: e instanceof Error ? e.message : String(e) })
          return false
        }
      },
      clearError: () => set({ error: null }),
      select: (selected) => set({ selected }),
    }
  })
}

export const simStore = createSimStore()

export function useSim<T>(selector: (s: SimStore) => T): T {
  return useStore(simStore, selector)
}
