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
})

const narration = () => within(screen.getByRole('region', { name: 'Narration' }))

async function startDemo() {
  render(<App />)
  await userEvent.click(screen.getByRole('button', { name: /^Demo/ }))
  simStore.getState().pause()
}

test('lists scenarios; picking one shows its narration and rebuilds the cluster', async () => {
  render(<App />)
  expect(screen.queryByRole('region', { name: 'Narration' })).toBeNull()
  await userEvent.click(screen.getByRole('button', { name: /^Demo/ }))
  expect(client.calls).toContain('scenario demo') // playback starts right after
  expect(narration().getByRole('heading')).toHaveTextContent('Demo · step 1 of 3')
  expect(narration().getByText('First.')).toBeInTheDocument()
  expect(screen.getAllByRole('button', { name: /^Node \d/ })).toHaveLength(3)
  expect(screen.getByRole('button', { name: /^Demo/ })).toHaveAttribute('aria-current', 'true')
  expect(screen.getByRole('slider', { name: 'Timeline' })).toHaveAttribute('aria-valuemax', '1000')
})

test('step buttons jump between narrated steps', async () => {
  await startDemo()
  expect(narration().getByRole('button', { name: 'Previous step' })).toBeDisabled()
  await userEvent.click(narration().getByRole('button', { name: 'Next step' }))
  expect(client.calls.at(-1)).toBe('seek 301') // just after, so the step's action shows
  expect(narration().getByText('Second.')).toBeInTheDocument()
  await userEvent.click(narration().getByRole('button', { name: 'Next step' })) // skips the silent step
  expect(client.calls.at(-1)).toBe('seek 801')
  expect(narration().getByRole('button', { name: 'Next step' })).toBeDisabled()
  await userEvent.click(narration().getByRole('button', { name: 'Previous step' }))
  expect(narration().getByText('Second.')).toBeInTheDocument()
})

test('acting mid-scenario warns that the narration may not match; exit returns to free play', async () => {
  await startDemo()
  await userEvent.click(screen.getByRole('button', { name: /^Node 2,/ }))
  await userEvent.click(screen.getByRole('button', { name: 'Crash' }))
  expect(narration().getByText(/may no longer match/)).toBeInTheDocument()
  await userEvent.click(narration().getByRole('button', { name: 'Exit scenario' }))
  expect(screen.queryByRole('region', { name: 'Narration' })).toBeNull()
})

test('replay rewinds to the start and plays', async () => {
  await startDemo()
  await userEvent.click(narration().getByRole('button', { name: 'Next step' }))
  const before = client.calls.length
  await userEvent.click(narration().getByRole('button', { name: 'Replay scenario' }))
  expect(client.calls[before]).toBe('seek 0') // playback may advance right after
  expect(simStore.getState().playing).toBe(true)
})
