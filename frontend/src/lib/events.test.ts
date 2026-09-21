import { expect, test } from 'vitest'
import type { SimEvent } from '@/client/types'
import { CATEGORY, describe, formatTime } from './events'

const ev = (over: Partial<SimEvent>): SimEvent => ({
  time: 0,
  type: 'became-leader',
  node: 3,
  term: 4,
  ...over,
})

test('describes events as sentences', () => {
  expect(describe(ev({}))).toBe('N3 became leader of term 4')
  expect(describe(ev({ type: 'vote-granted', peer: 2 }))).toBe('N3 voted for N2 in term 4')
  expect(describe(ev({ type: 'log-truncated', index: 5 }))).toBe('N3 discarded conflicting entries from #5')
  expect(describe(ev({ type: 'node-crashed' }))).toBe('N3 crashed')
})

test('every event type has a category', () => {
  expect(CATEGORY['commit-advanced']).toBe('log')
  expect(CATEGORY['node-paused']).toBe('fault')
})

test('formats virtual time in seconds', () => {
  expect(formatTime(1234)).toBe('1.234s')
})

test('every event type has a description', () => {
  const types = Object.keys(CATEGORY) as SimEvent['type'][]
  const texts = types.map((type) => describe(ev({ type, peer: 1, index: 7 })))
  expect(texts).toEqual([
    'N3 started an election for term 4',
    'N3 became leader of term 4',
    'N3 stepped down (term 4)',
    'N3 voted for N1 in term 4',
    'N3 discarded conflicting entries from #7',
    'N3 committed up to #7',
    'N3 crashed',
    'N3 restarted',
    'N3 paused',
    'N3 resumed',
  ])
})
