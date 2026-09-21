// @vitest-environment node
// Drives the real raft.wasm through the Engine, as the Web Worker does.
// Skipped until the module is built (`npm run wasm`); CI builds it first.
import { existsSync } from 'node:fs'
import { readFile } from 'node:fs/promises'
import { join } from 'node:path'
import { pathToFileURL } from 'node:url'
import { beforeAll, describe, expect, test } from 'vitest'
import type { SimEvent, SimState } from '../client/types'
import { Engine, type RaftSimApi } from './engine'

const pub = join(import.meta.dirname, '..', '..', 'public')
const built = existsSync(join(pub, 'raft.wasm'))

describe.skipIf(!built)('Engine with raft.wasm', () => {
  let engine: Engine
  const g = globalThis as unknown as {
    Go: new () => { importObject: WebAssembly.Imports; run(i: WebAssembly.Instance): void }
    raftSim: RaftSimApi
  }

  beforeAll(async () => {
    await import(pathToFileURL(join(pub, 'wasm_exec.js')).href)
    const go = new g.Go()
    const { instance } = await WebAssembly.instantiate(
      await readFile(join(pub, 'raft.wasm')),
      go.importObject,
    )
    go.run(instance)
    engine = new Engine(g.raftSim)
  })

  const parse = (f: { state: string; events: string; reset: boolean }) => ({
    state: JSON.parse(f.state) as SimState,
    events: JSON.parse(f.events) as SimEvent[],
    reset: f.reset,
  })

  test('streams only new events between frames', () => {
    expect(parse(engine.call('create', '{"seed": 2}')).reset).toBe(true)
    const a = parse(engine.call('advance', 1000))
    expect(a.reset).toBe(false)
    expect(a.events.some((e) => e.type === 'became-leader')).toBe(true)
    const b = parse(engine.call('advance', 1))
    expect(b.events.length).toBeLessThan(a.events.length)
    expect(b.state.time).toBe(1001)
  })

  test('resends the whole history after a rewind', () => {
    engine.call('create', '{"seed": 2}')
    engine.call('advance', 1000)
    const f = parse(engine.call('seek', 400))
    expect(f.reset).toBe(true)
    expect(f.events.every((e) => e.time <= 400)).toBe(true)
  })

  test('surfaces simulator errors', () => {
    engine.call('create', '')
    expect(() => engine.call('do', '{"kind":"crash","node":42}')).toThrow(/no node 42/)
    expect(() => engine.call('create', '{"size": 12}')).toThrow(/Size/)
  })
})
