import { useEffect } from 'react'
import { useSim } from '@/store/sim'

const TIMEOUT_MS = 4000

/** Briefly shows why the last action was rejected. */
export function ErrorToast() {
  const error = useSim((s) => s.error)
  const clear = useSim((s) => s.clearError)

  useEffect(() => {
    if (!error) return
    const t = setTimeout(clear, TIMEOUT_MS)
    return () => clearTimeout(t)
  }, [error, clear])

  if (!error) return null
  return (
    <div
      role="alert"
      className="fixed bottom-6 left-1/2 -translate-x-1/2 rounded-lg border border-destructive/40 bg-card px-4 py-2 font-mono text-sm text-destructive shadow-lg"
    >
      {error}
    </div>
  )
}
