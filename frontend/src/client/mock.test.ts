import { expect, test } from 'vitest'
import { MockClient } from './mock'

test('mock elects node 1 at 300ms and reports it once', async () => {
  const c = new MockClient()
  await c.create({ size: 3 })
  expect((await c.advance(200)).state.nodes[0].role).toBe('follower')
  const f = await c.advance(200)
  expect(f.state.nodes[0].role).toBe('leader')
  expect(f.events.map((e) => e.type)).toEqual(['became-leader'])
  expect((await c.advance(10)).events).toEqual([])
})

test('mock applies node actions and rejects proposals to followers', async () => {
  const c = new MockClient()
  await c.create()
  const f = await c.do({ kind: 'resume', node: 2 })
  expect(f.events[0].type).toBe('node-resumed')
  expect((await c.do({ kind: 'crash', node: 2 })).state.nodes[1].state).toBe('crashed')
  await expect(c.do({ kind: 'propose', node: 3, data: 'set x 1' })).rejects.toThrow(/not leader/)
  expect(c.calls).toContain('do {"kind":"crash","node":2}')
})
