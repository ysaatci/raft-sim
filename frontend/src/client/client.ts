import type { Action, Config, SimEvent, SimState } from './types'

/** What the UI gets back from every call: the new state plus new events. */
export interface Frame {
  state: SimState
  /** Events since the previous frame, or the whole history if `reset`. */
  events: SimEvent[]
  /** True when the history was rewound and the event list must be replaced. */
  reset: boolean
}

/**
 * A running simulation, whichever engine drives it: the WASM build in a Web
 * Worker, a live cluster over WebSocket, or a mock in tests. Every method
 * resolves with a fresh frame; a rejected action (e.g. proposing to a
 * follower) rejects with an Error carrying the simulator's message.
 */
export interface SimulationClient {
  create(config?: Partial<Config>): Promise<Frame>
  advance(ms: number): Promise<Frame>
  seek(time: number): Promise<Frame>
  do(action: Action): Promise<Frame>
  dispose(): void
}
