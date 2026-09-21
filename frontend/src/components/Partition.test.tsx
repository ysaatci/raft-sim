import { render, screen, within } from '@testing-library/react'
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
})

const link = (a: number, b: number) => screen.getByRole('button', { name: new RegExp(`^Link N${a}–N${b},`) })
const network = () => within(screen.getByRole('region', { name: 'Network' }))

test('clicking a link cuts it in both directions, clicking again restores it', async () => {
  render(<App />)
  expect(link(1, 2)).toHaveAccessibleName('Link N1–N2, connected')
  await userEvent.click(link(1, 2))
  expect(link(1, 2)).toHaveAccessibleName('Link N1–N2, cut')
  expect(client.calls.filter((c) => c.includes('set-link'))).toHaveLength(2)
  await userEvent.click(link(1, 2))
  expect(link(1, 2)).toHaveAccessibleName('Link N1–N2, connected')
})

test('a one-way cut is shown and restored by a click', async () => {
  const cut = { latencyMs: 20, jitterMs: 5, dropRate: 0, cut: true }
  await simStore.getState().act({ kind: 'set-link', node: 1, to: 2, link: cut })
  render(<App />)
  expect(link(1, 2)).toHaveAccessibleName('Link N1–N2, cut in one direction')
  await userEvent.click(link(1, 2))
  expect(link(1, 2)).toHaveAccessibleName('Link N1–N2, connected')
})

test('the partition tool splits the cluster and heal reconnects it', async () => {
  render(<App />)
  expect(network().getByRole('button', { name: /Heal all/ })).toBeDisabled()
  await userEvent.click(network().getByRole('button', { name: /Partition/ }))
  expect(network().getByRole('button', { name: 'Apply' })).toBeDisabled()

  await userEvent.click(screen.getByRole('button', { name: /^Node 1,/ }))
  expect(network().getByText('N1')).toBeInTheDocument()
  expect(network().getByText('N2 N3')).toBeInTheDocument()
  expect(screen.queryByRole('region', { name: 'Node 1' })).toBeNull() // picking, not selecting

  await userEvent.click(network().getByRole('button', { name: 'Apply' }))
  expect(client.calls.at(-1)).toBe('do {"kind":"partition","groups":[[1],[2,3]]}')
  expect(link(1, 2)).toHaveAccessibleName(/cut$/)
  expect(link(2, 3)).toHaveAccessibleName(/connected/)

  await userEvent.click(network().getByRole('button', { name: /Heal all/ }))
  expect(link(1, 2)).toHaveAccessibleName(/connected/)
})

test('cancel leaves the network untouched; one side cannot be everything', async () => {
  render(<App />)
  await userEvent.click(network().getByRole('button', { name: /Partition/ }))
  for (const id of [1, 2, 3])
    await userEvent.click(screen.getByRole('button', { name: new RegExp(`^Node ${id},`) }))
  expect(network().getByRole('button', { name: 'Apply' })).toBeDisabled()
  await userEvent.click(network().getByRole('button', { name: 'Cancel' }))
  expect(client.calls.some((c) => c.includes('partition'))).toBe(false)
})

test('links can be toggled from the keyboard', async () => {
  render(<App />)
  link(2, 3).focus()
  await userEvent.keyboard('{Enter}')
  expect(link(2, 3)).toHaveAccessibleName(/cut$/)
})
