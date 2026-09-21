import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, test } from 'vitest'
import { MockClient } from '@/client/mock'
import { simStore } from '@/store/sim'
import App from './App'

beforeEach(async () => {
  simStore.getState().pause()
  await simStore.getState().init(new MockClient(), { size: 3 })
})

test('renders the cluster once the simulation is loaded', () => {
  render(<App />)
  expect(screen.getByRole('heading', { name: 'Raft Simulator' })).toBeInTheDocument()
  expect(screen.getAllByRole('button', { name: /^Node \d/ })).toHaveLength(3)
})

test('play toggles to pause, and step advances the clock', async () => {
  render(<App />)
  await userEvent.click(screen.getByRole('button', { name: 'Step 10ms' }))
  expect(screen.getByLabelText('Simulated time')).toHaveTextContent('0.01s')
  await userEvent.click(screen.getByRole('button', { name: 'Play' }))
  expect(screen.getByRole('button', { name: 'Pause' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Step 10ms' })).toBeDisabled()
})

test('speed buttons select the playback speed', async () => {
  render(<App />)
  await userEvent.click(screen.getByRole('button', { name: '0.25×' }))
  expect(simStore.getState().speed).toBe(0.25)
  expect(screen.getByRole('button', { name: '0.25×' })).toHaveAttribute('aria-pressed', 'true')
})

test('renders a loading state before the simulator is ready', () => {
  simStore.setState({ state: null })
  render(<App />)
  expect(screen.getByText(/Loading simulator/)).toBeInTheDocument()
  expect(screen.getByLabelText('Safety checks: all passing')).toBeInTheDocument()
})
