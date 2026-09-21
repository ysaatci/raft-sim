import { render, screen, within } from '@testing-library/react'
import { expect, test } from 'vitest'
import type { NodeView } from '@/client/types'
import { Inspector } from './Inspector'

const base: NodeView = {
  id: 1,
  role: 'follower',
  term: 3,
  votedFor: 2,
  leader: 2,
  commit: 4,
  applied: 4,
  lastIndex: 5,
  lastTerm: 3,
  electionElapsed: 4,
  electionTimeout: 21,
  state: 'up',
  log: [],
  kv: { y: '2', x: '1' },
  inbox: 0,
}

const value = (label: string) => screen.getByText(label).nextElementSibling?.textContent

test('shows a follower’s vote, leader, log and timer', () => {
  render(<Inspector node={base} />)
  expect(value('voted for')).toBe('N2')
  expect(value('leader')).toBe('N2')
  expect(value('last log')).toBe('#5 (term 3)')
  expect(value('election timer')).toBe('4 / 21 ticks')
  expect(screen.queryByRole('list', { name: 'Replication' })).toBeNull()
})

test('lists the state machine sorted by key', () => {
  render(<Inspector node={base} />)
  const rows = within(screen.getByRole('table', { name: 'State machine' })).getAllByRole('row')
  expect(rows.map((r) => r.textContent)).toEqual(['x1', 'y2'])
})

test('shows a leader’s replication progress per follower', () => {
  render(
    <Inspector
      node={{ ...base, role: 'leader', next: { '1': 6, '2': 6, '3': 3 }, match: { '1': 5, '2': 5, '3': 2 } }}
    />,
  )
  expect(screen.queryByText('election timer')).toBeNull()
  const list = within(screen.getByRole('list', { name: 'Replication' }))
  expect(list.getAllByRole('listitem')).toHaveLength(2) // not itself
  expect(list.getByLabelText('N3 progress')).toHaveTextContent('2 / 3')
})

test('shows queued messages for a paused node and an empty state machine', () => {
  render(<Inspector node={{ ...base, state: 'paused', inbox: 7, kv: {} }} />)
  expect(value('queued')).toBe('7 messages')
  expect(screen.getByText('empty')).toBeInTheDocument()
})
