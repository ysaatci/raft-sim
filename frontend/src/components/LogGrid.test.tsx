import { render, screen, within } from '@testing-library/react'
import { expect, test } from 'vitest'
import type { NodeView } from '@/client/types'
import { MockClient } from '@/client/mock'
import { LogGrid } from './LogGrid'

async function nodes(): Promise<NodeView[]> {
  const ns = (await new MockClient().create({ size: 3 })).state.nodes
  const log = [
    { term: 1, index: 1, data: '' },
    { term: 1, index: 2, data: 'set x 1' },
    { term: 2, index: 3, data: 'set y 2' },
  ]
  ns[0] = { ...ns[0], log, lastIndex: 3, commit: 2 }
  ns[1] = { ...ns[1], log: log.slice(0, 2), lastIndex: 2, commit: 2 }
  ns[2] = { ...ns[2], state: 'crashed' }
  return ns
}

test('shows one column per index and marks committed entries', async () => {
  render(<LogGrid nodes={await nodes()} />)
  expect(screen.getAllByRole('columnheader').map((h) => h.textContent)).toEqual(['', '1', '2', '3'])

  const row = within(screen.getByRole('row', { name: 'Node 1 log' }))
  const cells = row.getAllByRole('cell').filter((c) => c.textContent)
  expect(cells.map((c) => c.textContent)).toEqual(['1', '1', '2'])
  expect(cells.map((c) => c.dataset.committed)).toEqual(['true', 'true', 'false'])
  expect(cells[1]).toHaveAttribute('title', '#2 term 1: set x 1')
  expect(cells[0]).toHaveAttribute('title', '#1 term 1: (no-op)')
  expect(cells[2].title).toMatch(/uncommitted/)
})

test('dims crashed nodes and explains an empty log', async () => {
  const ns = await nodes()
  const { rerender } = render(<LogGrid nodes={ns} />)
  expect(screen.getByRole('row', { name: 'Node 3 log' })).toHaveClass('opacity-50')
  rerender(<LogGrid nodes={ns.map((n) => ({ ...n, log: [], lastIndex: 0 }))} />)
  expect(screen.getByText(/No entries yet/)).toBeInTheDocument()
})

test('follows the end of a growing log unless the user scrolled back', async () => {
  const ns = await nodes()
  const { rerender } = render(<LogGrid nodes={ns} />)
  const scroller = screen.getByRole('table').parentElement!
  let scrollWidth = 1000
  Object.defineProperty(scroller, 'scrollWidth', { get: () => scrollWidth })
  Object.defineProperty(scroller, 'clientWidth', { value: 300 })

  const grow = (n: number) =>
    rerender(<LogGrid nodes={ns.map((x) => ({ ...x, lastIndex: n, log: [...x.log] }))} />)
  grow(10)
  expect(scroller.scrollLeft).toBe(1000)

  scroller.scrollLeft = 100 // the user scrolls back to read older entries
  scroller.dispatchEvent(new Event('scroll'))
  scrollWidth = 1200
  grow(11)
  expect(scroller.scrollLeft).toBe(100)
})
