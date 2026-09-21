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
  await simStore.getState().step(400) // node 1 elected at 300ms
  await simStore.getState().act({ kind: 'crash', node: 3 })
})

const feed = () => within(screen.getByRole('list', { name: 'Event list' }))

test('the event feed lists events newest first as sentences', () => {
  render(<App />)
  expect(
    feed()
      .getAllByRole('button')
      .map((b) => b.textContent),
  ).toEqual(['0.400sN3 crashed', '0.300sN1 became leader of term 1'])
})

test('filters hide categories of events', async () => {
  render(<App />)
  await userEvent.click(screen.getByRole('button', { name: 'Faults' }))
  expect(feed().getAllByRole('button')).toHaveLength(1)
  await userEvent.click(screen.getByRole('button', { name: 'Elections' }))
  expect(feed().getByText('Nothing yet.')).toBeInTheDocument()
})

test('clicking an event pauses and jumps to it', async () => {
  simStore.getState().play()
  render(<App />)
  await userEvent.click(feed().getByText('N1 became leader of term 1'))
  expect(simStore.getState().playing).toBe(false)
  expect(client.calls.at(-1)).toBe('seek 300')
})

test('the timeline spans the history and seeks when scrubbed', async () => {
  render(<App />)
  const slider = screen.getByRole('slider', { name: 'Timeline' })
  expect(slider).toHaveAttribute('aria-valuemax', '400')
  expect(slider).toHaveAttribute('aria-valuenow', '400')
  slider.focus()
  await userEvent.keyboard('{Home}')
  expect(client.calls.at(-1)).toBe('seek 0')
  expect(screen.getByText('0.400s')).toBeInTheDocument() // horizon stays
})
