import { render } from '@testing-library/react'
import { afterEach, expect, test, vi } from 'vitest'
import { simStore } from './sim'
import { useSimLoop } from './useSimLoop'

function Loop() {
  useSimLoop()
  return null
}

afterEach(() => vi.unstubAllGlobals())

test('ticks the store with real elapsed time every animation frame, and stops on unmount', () => {
  const frames: FrameRequestCallback[] = []
  vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => frames.push(cb))
  vi.stubGlobal('cancelAnimationFrame', vi.fn())
  vi.spyOn(performance, 'now').mockReturnValue(1000)
  const tick = vi.spyOn(simStore.getState(), 'tick').mockResolvedValue()
  simStore.setState({ tick })

  const { unmount } = render(<Loop />)
  frames.shift()!(1016)
  frames.shift()!(1050)
  expect(tick.mock.calls).toEqual([[16], [34]])

  unmount()
  expect(cancelAnimationFrame).toHaveBeenCalled()
})
