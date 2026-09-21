import type { Scenario } from '@/client/types'

export interface Narration {
  /** Index of the step whose narration is showing. */
  step: number
  text: string
}

/** The narration showing at virtual time `time`: the last non-empty one so far. */
export function narrationAt(sc: Scenario, time: number): Narration {
  let current: Narration = { step: 0, text: sc.steps[0]?.narration ?? '' }
  sc.steps.forEach((s, i) => {
    if (s.at <= time && s.narration) current = { step: i, text: s.narration }
  })
  return current
}

/** Steps that carry narration: the ones a reader can jump between. */
export function narratedSteps(sc: Scenario) {
  return sc.steps.map((s, i) => ({ ...s, index: i })).filter((s) => s.narration)
}
