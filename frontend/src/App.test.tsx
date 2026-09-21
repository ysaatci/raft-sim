import { render, screen } from '@testing-library/react'
import { expect, test } from 'vitest'
import App from './App'

test('renders the title and role legend', () => {
  render(<App />)
  expect(screen.getByRole('heading', { name: 'Raft Simulator' })).toBeInTheDocument()
  expect(screen.getByText('Leader')).toBeInTheDocument()
})
