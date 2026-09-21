import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { MockClient } from '@/client/mock'
import { simStore } from '@/store/sim'
import { ShareButton } from './ShareButton'

beforeEach(async () => {
  await simStore.getState().init(new MockClient())
  await simStore.getState().step(250)
})

afterEach(() => history.replaceState(null, '', '/'))

test('copies a link to this moment and puts it in the address bar', async () => {
  const writeText = vi.fn().mockResolvedValue(undefined)
  Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
  render(<ShareButton />)
  await userEvent.click(screen.getByRole('button', { name: 'Copy link to this moment' }))
  expect(location.hash).toMatch(/^#run=.*&t=250$/)
  expect(writeText).toHaveBeenCalledWith(location.href)
  expect(screen.getByRole('button', { name: 'Link copied' })).toBeInTheDocument()
})
