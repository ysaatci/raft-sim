import { render, screen } from '@testing-library/react'
import { expect, test, vi } from 'vitest'
import { MockClient } from '@/client/mock'
import { simStore } from '@/store/sim'
import App from '@/App'
import { ErrorBoundary } from './ErrorBoundary'

function Boom(): never {
  throw new Error('kaboom')
}

test('a render error shows a message and a reload button instead of a blank page', () => {
  vi.spyOn(console, 'error').mockImplementation(() => {})
  render(
    <ErrorBoundary>
      <Boom />
    </ErrorBoundary>,
  )
  expect(screen.getByRole('alert')).toHaveTextContent('Something went wrong')
  expect(screen.getByRole('alert')).toHaveTextContent('kaboom')
  expect(screen.getByRole('button', { name: 'Reload' })).toBeInTheDocument()
})

test('a simulator that fails to start is reported', async () => {
  const client = new MockClient()
  client.create = () => Promise.reject(new Error('simulator failed to start: 404'))
  await simStore.getState().init(client)
  render(<App />)
  expect(screen.getByRole('alert')).toHaveTextContent('The simulator could not start')
  expect(screen.getByRole('alert')).toHaveTextContent('404')
})
