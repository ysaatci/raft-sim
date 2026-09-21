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

const panel = () => within(screen.getByRole('region', { name: 'Network' }))

test('shows the current network conditions', () => {
  render(<App />)
  expect(panel().getByRole('slider', { name: 'Latency' })).toHaveAttribute('aria-valuenow', '20')
  expect(panel().getByText('±5 ms')).toBeInTheDocument()
  expect(panel().getByText('0%')).toBeInTheDocument()
})

test('a slider changes every link, keeping cuts', async () => {
  await simStore.getState().act({ kind: 'partition', groups: [[1], [2, 3]] })
  render(<App />)
  panel().getByRole('slider', { name: 'Packet loss' }).focus()
  await userEvent.keyboard('{ArrowRight}{ArrowRight}')
  expect(client.calls.at(-1)).toBe(
    'do {"kind":"set-network","link":{"latencyMs":20,"jitterMs":5,"dropRate":0.02,"cut":false}}',
  )
  expect(simStore.getState().state?.links.find((l) => l.from === 1 && l.to === 2)?.cut).toBe(true)
})

test('a single link can be tuned in both directions', async () => {
  render(<App />)
  await userEvent.selectOptions(panel().getByRole('combobox'), 'N1 ↔ N3')
  panel().getByRole('slider', { name: 'Latency' }).focus()
  await userEvent.keyboard('{ArrowRight}')
  const links = simStore.getState().state!.links
  expect(links.find((l) => l.from === 1 && l.to === 3)?.latencyMs).toBe(21)
  expect(links.find((l) => l.from === 3 && l.to === 1)?.latencyMs).toBe(21)
  expect(links.find((l) => l.from === 1 && l.to === 2)?.latencyMs).toBe(20)
})
