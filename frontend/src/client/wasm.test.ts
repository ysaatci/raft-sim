import { expect, test } from 'vitest'
import type { Request, Response } from '../worker/protocol'
import { WasmClient, type WorkerLike } from './wasm'

/** A fake worker that answers asynchronously, like the real one. */
function fakeWorker(answer: (req: Request) => Response) {
  const w: WorkerLike & { sent: Request[]; terminated: boolean } = {
    sent: [],
    terminated: false,
    onmessage: null,
    postMessage(req: Request) {
      this.sent.push(req)
      if (req.method !== 'seek') {
        queueMicrotask(() => w.onmessage?.({ data: answer(req) } as MessageEvent<Response>))
      }
    },
    terminate() {
      this.terminated = true
    },
  }
  return w
}

const frame = (time: number) => ({
  state: JSON.stringify({ time }),
  events: JSON.stringify([{ time, type: 'became-leader', node: 1, term: 1 }]),
  reset: false,
})

test('parses frames and matches responses to requests', async () => {
  const w = fakeWorker((req) => ({ id: req.id, frame: frame(Number('arg' in req ? req.arg : 0)) }))
  const c = new WasmClient(w)
  const [a, b] = await Promise.all([c.advance(10), c.advance(20)])
  expect(a.state.time).toBe(10)
  expect(b.state.time).toBe(20)
  expect(a.events[0].type).toBe('became-leader')
  expect(w.sent.map((r) => r.method)).toEqual(['advance', 'advance'])
})

test('serializes arguments', async () => {
  const w = fakeWorker((req) => ({ id: req.id, frame: frame(0) }))
  const c = new WasmClient(w)
  await c.create({ size: 3 })
  await c.create()
  await c.do({ kind: 'crash', node: 2 })
  expect(w.sent.map((r) => ('arg' in r ? r.arg : undefined))).toEqual([
    '{"size":3}',
    '',
    '{"kind":"crash","node":2}',
  ])
})

test('rejects with the simulator error message', async () => {
  const w = fakeWorker((req) => ({ id: req.id, error: 'raft: not leader, try node 3' }))
  await expect(new WasmClient(w).do({ kind: 'propose', node: 1, data: 'x' })).rejects.toThrow(
    'raft: not leader, try node 3',
  )
})

test('dispose terminates the worker and rejects pending calls', async () => {
  const w = fakeWorker((req) => ({ id: req.id, frame: frame(0) }))
  const c = new WasmClient(w)
  const pending = c.seek(5) // the fake never answers seeks
  c.dispose()
  expect(w.terminated).toBe(true)
  await expect(pending).rejects.toThrow('disposed')
})

test('a worker that fails to start rejects pending calls instead of hanging', async () => {
  const w = fakeWorker(() => {
    throw new Error('unreachable')
  })
  w.postMessage = () => {} // never answers
  const c = new WasmClient(w)
  const pending = c.create()
  w.onerror?.({ message: 'Failed to fetch raft.wasm' } as ErrorEvent)
  await expect(pending).rejects.toThrow('simulator failed to start: Failed to fetch raft.wasm')
})
