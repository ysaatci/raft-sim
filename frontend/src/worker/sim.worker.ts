// Hosts the WebAssembly simulator off the main thread.
import { Engine, type RaftSimApi } from './engine'
import type { Request, Response } from './protocol'

// Declared by hand: the webworker lib clashes with the DOM types used elsewhere.
declare const self: {
  Go: new () => GoRuntime
  raftSim: RaftSimApi
  onmessage: ((e: MessageEvent<Request>) => void) | null
  postMessage(res: Response): void
}

interface GoRuntime {
  importObject: WebAssembly.Imports
  run(instance: WebAssembly.Instance): Promise<void>
}

const base = import.meta.env.BASE_URL

const ready: Promise<Engine> = (async () => {
  await import(/* @vite-ignore */ `${base}wasm_exec.js`)
  const go = new self.Go()
  const { instance } = await WebAssembly.instantiateStreaming(fetch(`${base}raft.wasm`), go.importObject)
  void go.run(instance) // resolves only if the Go program exits
  return new Engine(self.raftSim)
})()

self.onmessage = async (e: MessageEvent<Request>) => {
  const req = e.data
  const { id } = req
  let res: Response
  try {
    const engine = await ready
    if ('arg' in req) res = { id, frame: engine.call(req.method, req.arg) }
    else res = { id, data: req.method === 'scenarios' ? engine.scenarios() : engine.actions() }
  } catch (err) {
    res = { id, error: err instanceof Error ? err.message : String(err) }
  }
  self.postMessage(res)
}
