// Smoke test for the compiled WebAssembly module: loads raft.wasm exactly
// as the browser will and drives it through the global `raftSim` API.
// Build first with `node scripts/build-wasm.mjs`, then run:
//   node --test backend/cmd/wasm/smoke.test.mjs
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { before, test } from 'node:test'
import { fileURLToPath, pathToFileURL } from 'node:url'

const publicDir = join(dirname(fileURLToPath(import.meta.url)), '..', '..', '..', 'frontend', 'public')
let sim

before(async () => {
  await import(pathToFileURL(join(publicDir, 'wasm_exec.js')))
  const go = new globalThis.Go()
  const { instance } = await WebAssembly.instantiate(await readFile(join(publicDir, 'raft.wasm')), go.importObject)
  go.run(instance) // never resolves: main blocks to keep serving calls
  sim = globalThis.raftSim
  assert.ok(sim, 'raftSim global not registered')
})

const state = () => JSON.parse(sim.state())
const leader = () => state().nodes.find((n) => n.role === 'leader' && n.state === 'up')?.id

test('queries before create return error objects', () => {
  assert.match(sim.state().error, /call create first/)
  assert.equal(sim.advance(10), 'bridge: no simulation; call create first')
})

test('create reports errors as strings', () => {
  assert.match(sim.create('{"size": 0}'), /Size/)
  assert.match(sim.create('{nope'), /invalid/)
  assert.equal(sim.create(''), null)
})

test('defaultConfig returns every field', () => {
  assert.deepEqual(Object.keys(JSON.parse(sim.defaultConfig())).sort(),
    ['electionTick', 'heartbeatTick', 'network', 'seed', 'size', 'tickMs'])
})

test('elects a leader and replicates a write', () => {
  sim.create('{"seed": 3}')
  assert.equal(sim.advance(1000), null)
  const id = leader()
  assert.ok(id, 'no leader after 1s')
  assert.equal(sim.do(JSON.stringify({ kind: 'propose', node: id, data: 'set color green' })), null)
  sim.advance(500)
  for (const n of state().nodes) assert.equal(n.kv.color, 'green', `node ${n.id}`)
})

test('crash, failover and rewind', () => {
  sim.create('{"seed": 4}')
  sim.advance(1000)
  const old = leader()
  sim.do(JSON.stringify({ kind: 'crash', node: old }))
  sim.advance(1000)
  const next = leader()
  assert.ok(next && next !== old, `leader ${next} after crashing ${old}`)
  assert.equal(state().nodes[old - 1].state, 'crashed')

  const total = JSON.parse(sim.events(0)).total
  assert.equal(sim.seek(500), null)
  assert.equal(state().time, 500)
  assert.equal(state().nodes[old - 1].state, 'up')
  assert.ok(JSON.parse(sim.events(0)).total < total, 'history not rewound')
  sim.seek(2000)
  assert.equal(leader(), next, 'replay diverged from the original run')
})

test('runs fast enough for real-time playback', () => {
  sim.create('{"seed": 5}')
  const start = performance.now()
  for (let i = 0; i < 600; i++) {
    sim.advance(100)
    if (i % 10 === 0) state()
  }
  const ms = performance.now() - start
  console.log(`60s of virtual time with 60 state snapshots: ${ms.toFixed(0)}ms`)
  assert.ok(ms < 5000, `too slow: ${ms}ms`)
  assert.deepEqual(state().violations, [])
})
