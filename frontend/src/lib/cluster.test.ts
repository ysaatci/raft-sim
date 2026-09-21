import { expect, test } from 'vitest'
import type { LinkView } from '@/client/types'
import { linkStatus, termColor } from './cluster'

const link = (from: number, to: number, cut: boolean): LinkView => ({
  from,
  to,
  cut,
  latencyMs: 20,
  jitterMs: 0,
  dropRate: 0,
})

test('link status looks at both directions', () => {
  expect(linkStatus([link(1, 2, false), link(2, 1, false)], 1, 2)).toBe('up')
  expect(linkStatus([link(1, 2, true), link(2, 1, true)], 2, 1)).toBe('cut')
  expect(linkStatus([link(1, 2, true), link(2, 1, false)], 1, 2)).toBe('one-way')
  expect(linkStatus([], 1, 2)).toBe('up')
})

test('term colors are stable and differ between neighbouring terms', () => {
  expect(termColor(3)).toBe(termColor(3))
  expect(termColor(3)).not.toBe(termColor(4))
})
