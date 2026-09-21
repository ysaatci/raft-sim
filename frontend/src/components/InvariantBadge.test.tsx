import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, test } from 'vitest'
import { MockClient } from '@/client/mock'
import { simStore } from '@/store/sim'
import { InvariantBadge } from './InvariantBadge'

beforeEach(async () => {
  await simStore.getState().init(new MockClient())
})

test('shows a passing badge that names the checked properties', () => {
  render(<InvariantBadge />)
  const badge = screen.getByLabelText('Safety checks: all passing')
  expect(badge).toHaveTextContent('Safe')
  expect(badge.title).toMatch(/Leader Completeness/)
})

test('turns red and lists violations on demand', async () => {
  render(<InvariantBadge />)
  act(() =>
    simStore.setState((s) => ({
      state: {
        ...s.state!,
        violations: [{ time: 1500, invariant: 'Election Safety', detail: 'nodes 1 and 2 both led term 3' }],
      },
    })),
  )
  const button = screen.getByRole('button', { name: /1 safety violation$/ })
  expect(screen.queryByRole('list', { name: 'Violations' })).toBeNull()
  await userEvent.click(button)
  expect(screen.getByRole('list', { name: 'Violations' })).toHaveTextContent(
    'Election Safety at 1.500snodes 1 and 2 both led term 3',
  )
})
