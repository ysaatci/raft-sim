import { expect, test } from 'vitest'
import { ringPositions } from './geometry'

test('places node 1 at the top and spaces nodes evenly', () => {
  const [a, b, c, d] = ringPositions(4, { x: 0, y: 0 }, 10)
  expect(a.x).toBeCloseTo(0)
  expect(a.y).toBeCloseTo(-10)
  expect(b.x).toBeCloseTo(10)
  expect(c.y).toBeCloseTo(10)
  expect(d.x).toBeCloseTo(-10)
})
