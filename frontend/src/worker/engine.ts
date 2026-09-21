// Runs next to the WASM module (in the Web Worker) and turns the raw
// `raftSim` API into frames. Kept free of Worker globals so it can be
// tested directly in Node.

/** The global object registered by backend/cmd/wasm. */
export interface RaftSimApi {
  defaultConfig(): string
  create(configJSON: string): string | null
  advance(ms: number): string | null
  seek(t: number): string | null
  do(actionJSON: string): string | null
  state(): string | { error: string }
  events(since: number): string | { error: string }
  actions(): string | { error: string }
  scenarios(): string
  loadScenario(id: string): string | null
  replay(configJSON: string, actionsJSON: string): string | null
}

/** A frame as sent to the main thread: JSON strings, parsed there. */
export interface RawFrame {
  state: string
  events: string
  reset: boolean
}

export type Method = 'create' | 'scenario' | 'replay' | 'advance' | 'seek' | 'do'

export class Engine {
  private api: RaftSimApi
  private cursor = 0 // how many history events the main thread has

  constructor(api: RaftSimApi) {
    this.api = api
  }

  call(method: Method, arg: string | number): RawFrame {
    let err: string | null
    let reset = false
    switch (method) {
      case 'create':
        err = this.api.create(arg as string)
        reset = true
        break
      case 'scenario':
        err = this.api.loadScenario(arg as string)
        reset = true
        break
      case 'replay': {
        const { config, actions } = JSON.parse(arg as string)
        err = this.api.replay(JSON.stringify(config), JSON.stringify(actions))
        reset = true
        break
      }
      case 'advance':
        err = this.api.advance(arg as number)
        break
      case 'seek': // may rewind the history, so the main thread gets it all again
        err = this.api.seek(arg as number)
        reset = true
        break
      case 'do':
        err = this.api.do(arg as string)
        break
    }
    if (err !== null) throw new Error(err)
    return this.frame(reset)
  }

  /** The recorded timeline, as JSON. */
  actions(): string {
    return unwrap(this.api.actions())
  }

  /** The built-in scenarios, as JSON. */
  scenarios(): string {
    return this.api.scenarios()
  }

  private frame(reset: boolean): RawFrame {
    if (reset) this.cursor = 0
    const state = unwrap(this.api.state())
    const page = JSON.parse(unwrap(this.api.events(this.cursor))) as { total: number; events: unknown[] }
    this.cursor = page.total
    return { state, events: JSON.stringify(page.events), reset }
  }
}

function unwrap(v: string | { error: string }): string {
  if (typeof v !== 'string') throw new Error(v.error)
  return v
}
