import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, test } from 'vitest'
import { MockClient } from '@/client/mock'
import { simStore } from '@/store/sim'
import App from '@/App'

let client: MockClient

beforeEach(async () => {
  client = new MockClient()
  simStore.getState().pause()
  await simStore.getState().init(client, { size: 3 })
  await simStore.getState().step(400) // node 1 leads
})

const node = (id: number) => screen.getByRole('button', { name: new RegExp(`^Node ${id},`) })

test('clicking a node opens its panel, clicking again closes it', async () => {
  render(<App />)
  await userEvent.click(node(2))
  expect(screen.getByRole('region', { name: 'Node 2' })).toBeInTheDocument()
  await userEvent.click(node(2))
  expect(screen.queryByRole('region', { name: 'Node 2' })).toBeNull()
})

test('crash and restart a node', async () => {
  render(<App />)
  await userEvent.click(node(2))
  await userEvent.click(screen.getByRole('button', { name: 'Crash' }))
  expect(node(2)).toHaveAccessibleName(/crashed/)
  expect(screen.getByRole('button', { name: /Send write/ })).toBeDisabled()
  await userEvent.click(screen.getByRole('button', { name: 'Restart' }))
  expect(client.calls.slice(-2)).toEqual(['do {"kind":"crash","node":2}', 'do {"kind":"restart","node":2}'])
})

test('pause and resume a node', async () => {
  render(<App />)
  await userEvent.click(node(3))
  await userEvent.click(screen.getByRole('button', { name: 'Pause' }))
  expect(node(3)).toHaveAccessibleName(/paused/)
  await userEvent.click(screen.getByRole('button', { name: 'Resume' }))
  expect(node(3)).toHaveAccessibleName(/follower/)
})

test('writes go to the selected node; a follower rejects them with a toast', async () => {
  render(<App />)
  await userEvent.click(node(1))
  await userEvent.click(screen.getByRole('button', { name: /Send write set x 1/ }))
  expect(client.calls.at(-1)).toBe('do {"kind":"propose","node":1,"data":"set x 1"}')
  expect(screen.getByRole('button', { name: /Send write set y 2/ })).toBeInTheDocument()

  await userEvent.click(node(2))
  await userEvent.click(screen.getByRole('button', { name: /Send write/ }))
  expect(screen.getByRole('alert')).toHaveTextContent('not leader')
  // A rejected write is not counted: the next attempt sends the same command.
  expect(screen.getByRole('button', { name: /Send write set y 2/ })).toBeInTheDocument()
})
