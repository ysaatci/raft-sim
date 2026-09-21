import { render, screen } from '@testing-library/react'
import { expect, test } from 'vitest'
import type { Flight } from '@/client/types'
import { MockClient } from '@/client/mock'
import { Packets } from './Packets'

const flights: Flight[] = [
  { id: 1, msg: { type: 'RequestVote', from: 1, to: 2, term: 2 }, sentAt: 0, deliverAt: 20 },
  {
    id: 2,
    msg: {
      type: 'AppendEntries',
      from: 1,
      to: 3,
      term: 2,
      entries: [{ term: 2, index: 1, data: 'set x 1' }],
    },
    sentAt: 0,
    deliverAt: 20,
  },
  {
    id: 3,
    msg: { type: 'AppendEntriesResp', from: 3, to: 1, term: 2, reject: true },
    sentAt: 0,
    deliverAt: 20,
  },
  { id: 4, msg: { type: 'AppendEntries', from: 1, to: 2, term: 2 }, sentAt: 0, deliverAt: 20, dropped: true },
  { id: 5, msg: { type: 'AppendEntries', from: 1, to: 2, term: 2 }, sentAt: 50, deliverAt: 70 }, // not sent yet
]

async function nodes() {
  const c = new MockClient()
  return (await c.create({ size: 3 })).state.nodes
}

test('draws in-flight messages with their type, entries and outcome', async () => {
  render(
    <svg>
      <Packets flights={flights} nodes={await nodes()} time={5} />
    </svg>,
  )
  const packets = screen.getAllByTestId('packet')
  expect(packets.map((p) => p.dataset.type)).toEqual([
    'RequestVote',
    'AppendEntries',
    'AppendEntriesResp',
    'AppendEntries',
  ])
  expect(packets[1]).toHaveTextContent('1') // entry count badge
  expect(packets[2].querySelector('title')).toHaveTextContent('rejected')
  expect(packets[3].dataset.dropped).toBe('true')
  expect(Number(packets[3].getAttribute('opacity'))).toBeLessThan(1)
})
