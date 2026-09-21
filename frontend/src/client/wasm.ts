import type { RawFrame } from '../worker/engine'
import type { Request, Response } from '../worker/protocol'
import type { Frame, SimulationClient } from './client'
import type { Action, Config, Scenario } from './types'

/** The part of Worker the client uses, so tests can substitute a fake. */
export type WorkerLike = Pick<Worker, 'postMessage' | 'terminate'> & {
  onmessage: ((e: MessageEvent<Response>) => void) | null
  onerror?: ((e: ErrorEvent) => void) | null
}

export function createSimWorker(): WorkerLike {
  return new Worker(new URL('../worker/sim.worker.ts', import.meta.url), { type: 'module' })
}

type Pending = { resolve: (res: Response) => void; reject: (e: Error) => void }

/** Talks to the WASM simulator running in a Web Worker. */
export class WasmClient implements SimulationClient {
  private worker: WorkerLike
  private nextId = 1
  private pending = new Map<number, Pending>()

  constructor(worker: WorkerLike = createSimWorker()) {
    this.worker = worker
    worker.onmessage = (e) => {
      const res = e.data
      const p = this.pending.get(res.id)
      if (!p) return
      this.pending.delete(res.id)
      if ('error' in res) p.reject(new Error(res.error))
      else p.resolve(res)
    }
    // A worker that fails to start (e.g. a missing file) would otherwise leave
    // every call hanging.
    worker.onerror = (e) => this.failAll(`simulator failed to start: ${e.message || 'worker error'}`)
  }

  create(config?: Partial<Config>) {
    return this.frame('create', config ? JSON.stringify(config) : '')
  }
  loadScenario(id: string) {
    return this.frame('scenario', id)
  }
  advance(ms: number) {
    return this.frame('advance', ms)
  }
  seek(time: number) {
    return this.frame('seek', time)
  }
  do(action: Action) {
    return this.frame('do', JSON.stringify(action))
  }
  async scenarios(): Promise<Scenario[]> {
    const res = await this.send({ id: this.nextId++, method: 'scenarios' })
    return 'data' in res ? JSON.parse(res.data) : []
  }
  dispose() {
    this.worker.terminate()
    this.failAll('simulation disposed')
  }

  private async frame(method: Exclude<Request['method'], 'scenarios'>, arg: string | number): Promise<Frame> {
    const res = await this.send({ id: this.nextId++, method, arg })
    if (!('frame' in res)) throw new Error(`unexpected reply to ${method}`)
    return parseFrame(res.frame)
  }

  private send(req: Request): Promise<Response> {
    return new Promise((resolve, reject) => {
      this.pending.set(req.id, { resolve, reject })
      this.worker.postMessage(req)
    })
  }

  private failAll(message: string) {
    for (const p of this.pending.values()) p.reject(new Error(message))
    this.pending.clear()
  }
}

function parseFrame(raw: RawFrame): Frame {
  return { state: JSON.parse(raw.state), events: JSON.parse(raw.events), reset: raw.reset }
}
