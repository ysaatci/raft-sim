import { expect, test } from 'vitest'
import type { Config } from '@/client/types'
import { decodeLink, encodeLink } from './share'

const config: Config = {
  size: 3,
  seed: 7,
  tickMs: 10,
  electionTick: 15,
  heartbeatTick: 5,
  network: { latencyMs: 20, jitterMs: 5, dropRate: 0, cut: false },
}

test('scenario links are short and readable', () => {
  const frag = encodeLink({ scenario: 'split-vote', time: 320.4 })
  expect(frag).toBe('scenario=split-vote&t=320')
  expect(decodeLink('#' + frag)).toEqual({ scenario: 'split-vote', time: 320 })
})

test('free-play links round-trip the config and every action', () => {
  const link = {
    config,
    actions: [
      { at: 400, kind: 'propose' as const, node: 1, data: 'set name Zoë 🚀' },
      { at: 900, kind: 'partition' as const, groups: [[1], [2, 3]] },
    ],
    time: 1500,
  }
  const frag = encodeLink(link)
  expect(frag).toMatch(/^run=[A-Za-z0-9_-]+&t=1500$/) // URL-safe, no padding
  expect(decodeLink(frag)).toEqual(link)
})

test('rejects anything that is not a link', () => {
  for (const bad of ['', '#', 'run=@@@', 'run=' + btoa('{"c":1}'), 'scenario=x&t=-5', 'scenario=x&t=abc']) {
    expect(decodeLink(bad)).toBeNull()
  }
})
