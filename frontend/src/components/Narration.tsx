import { ChevronLeft, ChevronRight, RotateCcw, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { narratedSteps, narrationAt } from '@/lib/scenario'
import { useSim } from '@/store/sim'

/** The current scenario's caption, with controls to move between its steps. */
export function Narration() {
  const scenario = useSim((s) => s.scenario)
  const time = useSim((s) => s.state?.time ?? 0)
  const diverged = useSim((s) => s.diverged)
  const { seek, pause, play, reset } = useSim((s) => s)
  if (!scenario) return null

  const steps = narratedSteps(scenario)
  const now = narrationAt(scenario, time)
  const pos = steps.findIndex((s) => s.index === now.step)
  // Land just after the step so its action (e.g. the crash) has already happened.
  const jump = (i: number) => {
    pause()
    void seek(steps[i].at + 1)
  }

  return (
    <section
      aria-label="Narration"
      className="flex flex-col gap-2 rounded-xl border border-primary/20 bg-card p-4"
    >
      <header className="flex items-center justify-between gap-2">
        <h2 className="text-sm font-semibold">
          {scenario.title}{' '}
          <span className="font-normal text-muted-foreground">
            · step {pos + 1} of {steps.length}
          </span>
        </h2>
        <div className="flex gap-1">
          <Button
            size="icon-sm"
            variant="ghost"
            aria-label="Previous step"
            disabled={pos <= 0}
            onClick={() => jump(pos - 1)}
          >
            <ChevronLeft />
          </Button>
          <Button
            size="icon-sm"
            variant="ghost"
            aria-label="Next step"
            disabled={pos >= steps.length - 1}
            onClick={() => jump(pos + 1)}
          >
            <ChevronRight />
          </Button>
          <Button
            size="icon-sm"
            variant="ghost"
            aria-label="Replay scenario"
            onClick={async () => {
              await seek(0)
              play()
            }}
          >
            <RotateCcw />
          </Button>
          <Button size="icon-sm" variant="ghost" aria-label="Exit scenario" onClick={() => reset()}>
            <X />
          </Button>
        </div>
      </header>
      <p aria-live="polite" className="text-sm leading-relaxed">
        {now.text}
      </p>
      {diverged && (
        <p className="text-xs text-candidate">
          You changed the run, so the narration may no longer match. Replay to see the original.
        </p>
      )}
    </section>
  )
}
