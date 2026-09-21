import type { RawFrame } from '../worker/engine'
import type { Request, Response } from '../worker/protocol'
import type { Frame, SimulationClient } from './client'
import type { Action, Config } from './types'

/** The part of Worker the client uses, so tests can substitute a fake. */
export type WorkerLike = Pick<Worker, 'postMessage' | 'terminate'> & {
  onmessage: ((e: MessageEvent<Response>) => void) | null
  onerror?: ((e: ErrorEvent) => void) | null
}

export function createSimWorker(): WorkerLike {
  return new Worker(new URL('../worker/sim.worker.ts', import.meta.url), { type: 'module' })
}

/** Talks to the WASM simulator running in a Web Worker. */
export class WasmClient implements SimulationClient {
  private worker: WorkerLike
  private nextId = 1
  private pending = new Map<number, { resolve: (f: Frame) => void; reject: (e: Error) => void }>()

  constructor(worker: WorkerLike = createSimWorker()) {
    this.worker = worker
    worker.onmessage = (e) => {
      const res = e.data
      const p = this.pending.get(res.id)
      if (!p) return
      this.pending.delete(res.id)
      if ('error' in res) p.reject(new Error(res.error))
      else p.resolve(parseFrame(res.frame))
    }
    // A worker that fails to start (e.g. a missing file) would otherwise leave
    // every call hanging.
    worker.onerror = (e) => this.failAll(`simulator failed to start: ${e.message || 'worker error'}`)
  }

  create(config?: Partial<Config>) {
    return this.call('create', config ? JSON.stringify(config) : '')
  }
  advance(ms: number) {
    return this.call('advance', ms)
  }
  seek(time: number) {
    return this.call('seek', time)
  }
  do(action: Action) {
    return this.call('do', JSON.stringify(action))
  }
  dispose() {
    this.worker.terminate()
    this.failAll('simulation disposed')
  }

  private failAll(message: string) {
    for (const p of this.pending.values()) p.reject(new Error(message))
    this.pending.clear()
  }

  private call(method: Request['method'], arg: Request['arg']): Promise<Frame> {
    const id = this.nextId++
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject })
      this.worker.postMessage({ id, method, arg } satisfies Request)
    })
  }
}

function parseFrame(raw: RawFrame): Frame {
  return { state: JSON.parse(raw.state), events: JSON.parse(raw.events), reset: raw.reset }
}
