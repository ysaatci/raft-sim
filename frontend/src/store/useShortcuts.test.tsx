import { render } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, test } from 'vitest'
import { MockClient } from '@/client/mock'
import { simStore } from './sim'
import { useShortcuts } from './useShortcuts'

function Shortcuts() {
  useShortcuts()
  return <input aria-label="field" />
}

let client: MockClient
const s = () => simStore.getState()

beforeEach(async () => {
  client = new MockClient()
  await s().init(client, { size: 3 })
  s().pause()
  render(<Shortcuts />)
})

test('space toggles playback', async () => {
  await userEvent.keyboard(' ')
  expect(s().playing).toBe(true)
  await userEvent.keyboard(' ')
  expect(s().playing).toBe(false)
})

test('arrow right steps while paused', async () => {
  await userEvent.keyboard('{ArrowRight}')
  expect(client.calls.at(-1)).toBe('advance 10')
})

test('digits select nodes, escape deselects', async () => {
  await userEvent.keyboard('2')
  expect(s().selected).toBe(2)
  await userEvent.keyboard('9') // no such node
  expect(s().selected).toBe(2)
  await userEvent.keyboard('{Escape}')
  expect(s().selected).toBeNull()
})

test('digits pick nodes while drawing a partition; escape cancels it', async () => {
  s().startPartition()
  await userEvent.keyboard('13')
  expect(s().partitionDraft).toEqual([1, 3])
  await userEvent.keyboard('{Escape}')
  expect(s().partitionDraft).toBeNull()
})

test('typing in a field is not a shortcut', async () => {
  await userEvent.click(document.querySelector('input')!)
  await userEvent.keyboard(' 2')
  expect(s().playing).toBe(false)
  expect(s().selected).toBeNull()
})
