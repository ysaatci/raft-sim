import { createStore, useStore } from 'zustand'
import type { Frame, SimulationClient } from '@/client/client'
import type { Action, Config, SimEvent, SimState } from '@/client/types'

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

  init(client: SimulationClient, config?: Partial<Config>): Promise<void>
  play(): void
  pause(): void
  setSpeed(speed: number): void
  /** Called every animation frame with the real time elapsed. */
  tick(realMs: number): Promise<void>
  step(ms: number): Promise<void>
  seek(time: number): Promise<void>
  act(action: Action): Promise<void>
  clearError(): void
}

export function createSimStore() {
  let busy = false // a request is in flight; frames never overlap
  let carry = 0 // virtual time owed but not yet advanced

  return createStore<SimStore>()((set, get) => {
    const apply = (f: Frame) =>
      set((s) => ({
        state: f.state,
        events: f.reset ? f.events.slice(-MAX_EVENTS) : [...s.events, ...f.events].slice(-MAX_EVENTS),
      }))

    // Runs one request at a time; returns false if another is in flight.
    const run = async (req: (c: SimulationClient) => Promise<Frame>) => {
      const client = get().client
      if (!client || busy) return false
      busy = true
      try {
        apply(await req(client))
      } finally {
        busy = false
      }
      return true
    }

    return {
      client: null,
      state: null,
      events: [],
      playing: false,
      speed: DEFAULT_SPEED,
      error: null,

      async init(client, config) {
        get().client?.dispose()
        busy = false
        carry = 0
        set({ client, state: null, events: [], error: null })
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
        if (await run((c) => c.advance(ms))) carry -= ms
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
        } catch (e) {
          set({ error: e instanceof Error ? e.message : String(e) })
        }
      },
      clearError: () => set({ error: null }),
    }
  })
}

export const simStore = createSimStore()

export function useSim<T>(selector: (s: SimStore) => T): T {
  return useStore(simStore, selector)
}
