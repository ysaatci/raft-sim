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
  await simStore.getState().init(client)
  await simStore.getState().step(400)
})

const settings = () => within(screen.getByRole('group', { name: 'Cluster settings' }))
const lastCreate = () =>
  JSON.parse(
    client.calls
      .filter((c) => c.startsWith('create'))
      .at(-1)!
      .slice(7),
  )

test('applies size, seed and timings, restarting the simulation', async () => {
  render(<App />)
  await userEvent.click(screen.getByText('Cluster'))
  await userEvent.click(settings().getByRole('button', { name: '3' }))
  const [seed, heartbeat, election] = settings().getAllByRole('spinbutton')
  await userEvent.clear(seed)
  await userEvent.type(seed, '42')
  await userEvent.clear(heartbeat)
  await userEvent.type(heartbeat, '100')
  await userEvent.clear(election)
  await userEvent.type(election, '400')
  await userEvent.click(settings().getByRole('button', { name: 'Apply and restart' }))

  expect(lastCreate()).toMatchObject({ size: 3, seed: 42, heartbeatTick: 10, electionTick: 40 })
  expect(screen.getAllByRole('button', { name: /^Node \d/ })).toHaveLength(3)
  expect(screen.getByLabelText('Simulated time')).toHaveTextContent('0.00s')
})

test('rejects an election timeout shorter than the heartbeat', async () => {
  render(<App />)
  await userEvent.click(screen.getByText('Cluster'))
  const election = settings().getAllByRole('spinbutton')[2]
  await userEvent.clear(election)
  await userEvent.type(election, '20')
  expect(settings().getByRole('alert')).toHaveTextContent('longer than the heartbeat')
  expect(settings().getByRole('button', { name: 'Apply and restart' })).toBeDisabled()
})

test('reset restarts with the current configuration', async () => {
  render(<App />)
  await userEvent.click(screen.getByRole('button', { name: 'Reset' }))
  expect(lastCreate()).toMatchObject({ size: 5, seed: 1 })
  expect(screen.getByLabelText('Simulated time')).toHaveTextContent('0.00s')
})

test('an emptied field is reported instead of silently replaced', async () => {
  render(<App />)
  await userEvent.click(screen.getByText('Cluster'))
  await userEvent.clear(settings().getAllByRole('spinbutton')[1])
  expect(settings().getByRole('alert')).toHaveTextContent('heartbeat at least 10 ms')
})
