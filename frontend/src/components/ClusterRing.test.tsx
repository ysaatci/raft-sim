import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, test, vi } from 'vitest'
import { MockClient } from '@/client/mock'
import { ClusterRing } from './ClusterRing'

async function nodes() {
  const c = new MockClient()
  await c.create({ size: 3 })
  return (await c.advance(400)).state.nodes // node 1 leads
}

test('labels every node with its role and term', async () => {
  render(<ClusterRing nodes={await nodes()} />)
  expect(screen.getByRole('button', { name: 'Node 1, leader, term 1' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Node 2, follower, term 1' })).toBeInTheDocument()
})

test('shows election timers only for running non-leaders', async () => {
  const ns = await nodes()
  ns[2].state = 'crashed'
  render(<ClusterRing nodes={ns} />)
  expect(screen.queryByTestId('timer-1')).toBeNull()
  expect(screen.getByTestId('timer-2')).toBeInTheDocument()
  expect(screen.queryByTestId('timer-3')).toBeNull()
  expect(screen.getByRole('button', { name: /Node 3, crashed/ })).toBeInTheDocument()
})

test('reports clicks and keyboard selection', async () => {
  const onSelect = vi.fn()
  render(<ClusterRing nodes={await nodes()} selected={2} onSelect={onSelect} />)
  await userEvent.click(screen.getByRole('button', { name: /Node 3/ }))
  screen.getByRole('button', { name: /Node 1/ }).focus()
  await userEvent.keyboard('{Enter}')
  expect(onSelect.mock.calls).toEqual([[3], [1]])
  expect(screen.getByRole('button', { name: /Node 2/ })).toHaveAttribute('aria-pressed', 'true')
})
