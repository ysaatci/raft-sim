import { act, render, screen } from '@testing-library/react'
import { afterEach, expect, test, vi } from 'vitest'
import { simStore } from '@/store/sim'
import { ErrorToast } from './ErrorToast'

afterEach(() => vi.useRealTimers())

test('shows the last error and dismisses it after 4 seconds', () => {
  vi.useFakeTimers()
  render(<ErrorToast />)
  expect(screen.queryByRole('alert')).toBeNull()

  act(() => simStore.setState({ error: 'raft: not leader, try node 3' }))
  expect(screen.getByRole('alert')).toHaveTextContent('try node 3')

  act(() => vi.advanceTimersByTime(3999))
  expect(screen.getByRole('alert')).toBeInTheDocument()
  act(() => vi.advanceTimersByTime(1))
  expect(screen.queryByRole('alert')).toBeNull()
})
