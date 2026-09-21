import { expect, test } from 'vitest'
import type { Flight } from '@/client/types'
import { packetPosition } from './packets'

const flight = (over: Partial<Flight> = {}): Flight => ({
  id: 1,
  msg: { type: 'AppendEntries', from: 1, to: 2, term: 1 },
  sentAt: 100,
  deliverAt: 120,
  ...over,
})
const A = { x: 0, y: 0 }
const B = { x: 100, y: 0 }

test('moves from the sender edge to the receiver edge on its lane', () => {
  const start = packetPosition(flight(), 100, A, B, 10)!
  const mid = packetPosition(flight(), 110, A, B, 10)!
  const end = packetPosition(flight(), 120, A, B, 10)!
  expect(start).toMatchObject({ x: 10, y: 7, opacity: 1 })
  expect(mid.x).toBeCloseTo(50)
  expect(end.x).toBeCloseTo(90)
})

test('the reverse direction uses the other lane', () => {
  const p = packetPosition(flight(), 110, B, A, 10)!
  expect(p.y).toBe(-7)
})

test('is hidden before sending and after arrival', () => {
  expect(packetPosition(flight(), 99, A, B, 10)).toBeNull()
  expect(packetPosition(flight(), 121, A, B, 10)).toBeNull()
})

test('dropped packets fade out and vanish halfway', () => {
  const d = flight({ dropped: true })
  expect(packetPosition(d, 100, A, B, 10)!.opacity).toBe(1)
  expect(packetPosition(d, 105, A, B, 10)!.opacity).toBeCloseTo(0.5)
  expect(packetPosition(d, 111, A, B, 10)).toBeNull()
})
