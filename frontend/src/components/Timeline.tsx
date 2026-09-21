import { useState } from 'react'
import { Slider } from '@/components/ui/slider'
import { formatTime } from '@/lib/events'
import { useSim } from '@/store/sim'

const MARKERS = {
  'became-leader': 'var(--leader)',
  'node-crashed': 'var(--crashed)',
  'node-paused': 'var(--paused)',
} as const

/** Scrub back and forth through the simulation's history. */
export function Timeline() {
  const time = useSim((s) => s.state?.time ?? 0)
  const horizon = useSim((s) => s.horizon)
  const events = useSim((s) => s.events)
  const { seek, pause } = useSim((s) => s)
  const [drag, setDrag] = useState<number | null>(null)

  const end = Math.max(horizon, 1)
  const markers = events.filter((e) => e.type in MARKERS)

  return (
    <div className="flex items-center gap-3 border-b px-4 py-2 sm:px-6">
      <span className="font-mono text-xs text-muted-foreground">0s</span>
      <div className="relative flex-1">
        <div aria-hidden className="pointer-events-none absolute inset-x-0 -top-1.5 h-1.5">
          {markers.map((e, i) => (
            <span
              key={i}
              className="absolute top-0 h-1.5 w-0.5 rounded-full"
              style={{
                left: `${(e.time / end) * 100}%`,
                background: MARKERS[e.type as keyof typeof MARKERS],
              }}
            />
          ))}
        </div>
        <Slider
          aria-label="Timeline"
          min={0}
          max={end}
          step={1}
          value={[drag ?? time]}
          onPointerDown={pause}
          onValueChange={([v]) => setDrag(v)}
          onValueCommit={async ([v]) => {
            await seek(v)
            setDrag(null)
          }}
        />
      </div>
      <span className="font-mono text-xs text-muted-foreground">{formatTime(horizon)}</span>
    </div>
  )
}
