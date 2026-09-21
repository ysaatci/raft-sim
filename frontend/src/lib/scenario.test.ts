import { expect, test } from 'vitest'
import { MOCK_SCENARIOS } from '@/client/mock'
import { narratedSteps, narrationAt } from './scenario'

const demo = MOCK_SCENARIOS[0]

test('shows the latest narration, keeping it through silent steps', () => {
  expect(narrationAt(demo, 0)).toEqual({ step: 0, text: 'First.' })
  expect(narrationAt(demo, 299).text).toBe('First.')
  expect(narrationAt(demo, 300)).toEqual({ step: 1, text: 'Second.' })
  expect(narrationAt(demo, 600)).toEqual({ step: 1, text: 'Second.' }) // step 2 is silent
  expect(narrationAt(demo, 5000)).toEqual({ step: 3, text: 'Third.' })
})

test('lists only narrated steps', () => {
  expect(narratedSteps(demo).map((s) => s.at)).toEqual([0, 300, 800])
})
