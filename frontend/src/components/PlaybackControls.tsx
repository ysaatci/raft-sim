import { Pause, Play, RotateCcw, StepForward } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { SPEEDS, useSim } from '@/store/sim'
import { cn } from '@/lib/utils'

const STEP_MS = 10 // one Raft tick at the default config

export function PlaybackControls() {
  const playing = useSim((s) => s.playing)
  const speed = useSim((s) => s.speed)
  const time = useSim((s) => s.state?.time ?? 0)
  const { play, pause, step, setSpeed, reset } = useSim((s) => s)

  return (
    <div className="flex flex-wrap items-center gap-2 sm:gap-3">
      <span
        className="w-16 text-right font-mono text-sm tabular-nums text-muted-foreground sm:w-24"
        aria-label="Simulated time"
      >
        {(time / 1000).toFixed(2)}s
      </span>
      <Button size="icon" onClick={playing ? pause : play} aria-label={playing ? 'Pause' : 'Play'}>
        {playing ? <Pause /> : <Play />}
      </Button>
      <Button
        size="icon"
        variant="outline"
        onClick={() => step(STEP_MS)}
        disabled={playing}
        aria-label="Step 10ms"
      >
        <StepForward />
      </Button>
      <Button size="icon" variant="outline" onClick={() => reset()} aria-label="Reset">
        <RotateCcw />
      </Button>
      <div className="flex rounded-lg border p-0.5" role="group" aria-label="Speed">
        {SPEEDS.map((s) => (
          <button
            key={s}
            onClick={() => setSpeed(s)}
            aria-pressed={speed === s}
            className={cn(
              'rounded-md px-2 py-1 font-mono text-xs text-muted-foreground transition-colors',
              speed === s && 'bg-secondary text-foreground',
            )}
          >
            {s}×
          </button>
        ))}
      </div>
    </div>
  )
}
