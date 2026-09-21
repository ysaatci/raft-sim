import { useState } from 'react'
import { CATEGORY, describe, formatTime, type EventCategory } from '@/lib/events'
import { cn } from '@/lib/utils'
import { useSim } from '@/store/sim'

const SHOWN = 150
const FILTERS: { key: EventCategory; label: string; color: string }[] = [
  { key: 'election', label: 'Elections', color: 'var(--leader)' },
  { key: 'fault', label: 'Faults', color: 'var(--crashed)' },
  { key: 'log', label: 'Log', color: 'var(--msg-append)' },
]

/** What happened, newest first. Click an event to jump to that moment. */
export function EventFeed() {
  const events = useSim((s) => s.events)
  const { seek, pause } = useSim((s) => s)
  // Commits happen constantly, so the log category starts hidden.
  const [on, setOn] = useState<Record<EventCategory, boolean>>({ election: true, fault: true, log: false })

  const shown = events
    .filter((e) => on[CATEGORY[e.type]])
    .slice(-SHOWN)
    .reverse()

  return (
    <section aria-label="Events" className="flex min-h-0 flex-col gap-3 rounded-xl border bg-card p-4">
      <header className="flex items-center justify-between gap-2">
        <h2 className="font-semibold">Events</h2>
        <div className="flex gap-1" role="group" aria-label="Filter events">
          {FILTERS.map((f) => (
            <button
              key={f.key}
              aria-pressed={on[f.key]}
              onClick={() => setOn((o) => ({ ...o, [f.key]: !o[f.key] }))}
              className={cn(
                'rounded-full border px-2 py-0.5 text-xs text-muted-foreground transition-colors',
                on[f.key] && 'bg-secondary text-foreground',
              )}
            >
              {f.label}
            </button>
          ))}
        </div>
      </header>
      <ol className="flex max-h-72 flex-col overflow-y-auto text-sm" aria-label="Event list">
        {shown.map((e, i) => (
          <li key={`${e.time}-${i}`}>
            <button
              onClick={() => {
                pause()
                void seek(e.time)
              }}
              className="flex w-full items-baseline gap-2 rounded px-1 py-0.5 text-left hover:bg-secondary/60"
            >
              <span className="font-mono text-xs tabular-nums text-muted-foreground">
                {formatTime(e.time)}
              </span>
              <span
                className="size-1.5 shrink-0 translate-y-[-1px] rounded-full"
                style={{ background: FILTERS.find((f) => f.key === CATEGORY[e.type])!.color }}
              />
              <span>{describe(e)}</span>
            </button>
          </li>
        ))}
        {shown.length === 0 && <li className="text-muted-foreground">Nothing yet.</li>}
      </ol>
    </section>
  )
}
