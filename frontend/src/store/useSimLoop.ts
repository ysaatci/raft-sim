import { useEffect } from 'react'
import { simStore } from './sim'

/** Drives the simulation from requestAnimationFrame while mounted. */
export function useSimLoop() {
  useEffect(() => {
    let last = performance.now()
    let frame = requestAnimationFrame(function loop(now) {
      void simStore.getState().tick(now - last)
      last = now
      frame = requestAnimationFrame(loop)
    })
    return () => cancelAnimationFrame(frame)
  }, [])
}
