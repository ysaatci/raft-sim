import { beforeEach, expect, test } from 'vitest'
import { MockClient } from '@/client/mock'
import { createSimStore } from './sim'

let store: ReturnType<typeof createSimStore>
let client: MockClient

beforeEach(async () => {
  store = createSimStore()
  client = new MockClient()
  await store.getState().init(client, { size: 3 })
})

const s = () => store.getState()

test('init creates the simulation and loads the first frame', () => {
  expect(client.calls).toEqual(['create {"size":3}'])
  expect(s().state?.nodes).toHaveLength(3)
  expect(s().playing).toBe(false)
})

test('tick does nothing while paused', async () => {
  await s().tick(1000)
  expect(client.calls).toHaveLength(1)
})

test('tick converts real time to virtual time at the current speed', async () => {
  s().play()
  s().setSpeed(0.1)
  await s().tick(16) // 1.6ms owed: advance 1, carry 0.6
  await s().tick(16) // 2.2ms owed: advance 2
  expect(client.calls.slice(1)).toEqual(['advance 1', 'advance 2'])
  expect(s().state?.time).toBe(3)
})

test('a slow frame never advances more than 50ms at once', async () => {
  s().play()
  s().setSpeed(1)
  await s().tick(5000)
  expect(client.calls.at(-1)).toBe('advance 50')
})

test('requests never overlap', async () => {
  s().play()
  s().setSpeed(1)
  await Promise.all([s().tick(10), s().tick(10), s().tick(10)])
  expect(client.calls.filter((c) => c.startsWith('advance'))).toHaveLength(1)
})

test('events accumulate and are replaced on reset', async () => {
  await s().step(400)
  expect(s().events.map((e) => e.type)).toEqual(['became-leader'])
  await s().act({ kind: 'crash', node: 2 })
  expect(s().events).toHaveLength(2)
  await s().seek(100)
  expect(s().events).toEqual([])
})

test('a rejected action is surfaced as an error and cleared by the next success', async () => {
  await s().act({ kind: 'propose', node: 2, data: 'set x 1' })
  expect(s().error).toMatch(/not leader/)
  await s().act({ kind: 'crash', node: 3 })
  expect(s().error).toBeNull()
})

test('re-init disposes the previous client', async () => {
  await s().init(new MockClient())
  expect(client.calls.at(-1)).toBe('dispose')
})

test('an action during an in-flight advance is queued, not dropped', async () => {
  s().play()
  s().setSpeed(1)
  const tick = s().tick(10)
  const act = s().act({ kind: 'crash', node: 2 })
  await Promise.all([tick, act])
  expect(client.calls.slice(1)).toEqual(['advance 10', 'do {"kind":"crash","node":2}'])
  expect(s().state?.nodes[1].state).toBe('crashed')
})

test('the horizon tracks the furthest time and moves back when acting in the past', async () => {
  await s().step(500)
  await s().seek(200)
  expect(s().state?.time).toBe(200)
  expect(s().horizon).toBe(500)
  await s().step(100) // replaying the recorded future keeps it
  expect(s().horizon).toBe(500)
  await s().act({ kind: 'crash', node: 2 }) // a new branch at 300
  expect(s().horizon).toBe(300)
})

test('reset starts over on the same client', async () => {
  await s().step(500)
  s().select(2)
  await s().reset({ size: 5, seed: 7 })
  expect(client.calls.at(-1)).toBe('create {"size":5,"seed":7}')
  expect(client.calls).not.toContain('dispose')
  expect(s().state?.time).toBe(0)
  expect(s().state?.nodes).toHaveLength(5)
  expect(s().horizon).toBe(0)
  expect(s().selected).toBeNull()
  await s().reset() // keeps the current config
  expect(JSON.parse(client.calls.at(-1)!.slice('create '.length)).seed).toBe(7)
})

test('reset during an in-flight advance still starts from zero', async () => {
  s().play()
  s().setSpeed(1)
  const tick = s().tick(40)
  const reset = s().reset()
  await Promise.all([tick, reset])
  expect(s().state?.time).toBe(0)
  expect(s().horizon).toBe(0)
})

test('init fetches the scenarios; loading one plays its whole script', async () => {
  expect(s().scenarios.map((x) => x.id)).toEqual(['demo', 'other'])
  await s().loadScenario('demo')
  expect(client.calls.at(-1)).toBe('scenario demo')
  expect(s().scenario?.id).toBe('demo')
  expect(s().state?.nodes).toHaveLength(3)
  expect(s().horizon).toBe(1000)
  expect(s().playing).toBe(true)
})

test('acting during a scenario marks it diverged; reset returns to free play', async () => {
  await s().loadScenario('demo')
  await s().act({ kind: 'crash', node: 2 })
  expect(s().diverged).toBe(true)
  await s().reset()
  expect(s().scenario).toBeNull()
  expect(s().diverged).toBe(false)
})

test('a free-play link replays the recorded actions and seeks to the shared time', async () => {
  await s().step(400)
  await s().act({ kind: 'crash', node: 2 })
  await s().step(100)
  const link = await s().shareLink()
  expect(link).toMatch(/^run=.*&t=500$/)

  const other = new MockClient()
  await s().init(other)
  expect(await s().openLink('#' + link)).toBe(true)
  const replayed = other.calls.find((c) => c.startsWith('replay'))!
  expect(JSON.parse(replayed.slice(7)).actions).toEqual([{ kind: 'crash', node: 2, at: 400 }])
  expect(other.calls.at(-1)).toBe('seek 500')
  expect(s().playing).toBe(false)
  expect(s().horizon).toBe(500)
})

test('a scenario link loads the scenario; after diverging it becomes a run link', async () => {
  await s().loadScenario('demo')
  s().pause()
  await s().step(300)
  expect(await s().shareLink()).toBe('scenario=demo&t=300')
  await s().act({ kind: 'crash', node: 1 })
  expect(await s().shareLink()).toMatch(/^run=/)

  await s().init(new MockClient())
  expect(await s().openLink('scenario=demo&t=300')).toBe(true)
  expect(s().scenario?.id).toBe('demo')
  expect(s().state?.time).toBe(300)
})

test('an invalid link reports an error', async () => {
  expect(await s().openLink('#run=garbage')).toBe(false)
  expect(await s().openLink('#scenario=missing')).toBe(false)
  expect(s().error).toMatch(/could not be opened/)
})
