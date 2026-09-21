import { ShieldAlert, ShieldCheck } from 'lucide-react'
import { useState } from 'react'
import { formatTime } from '@/lib/events'
import { NONE, useSim } from '@/store/sim'

const PROPERTIES = [
  'Election Safety: at most one leader per term',
  'Log Matching: logs that agree on an entry agree on everything before it',
  'Leader Completeness: committed entries are in every later leader’s log',
  'State Machine Safety: no two nodes apply different commands at an index',
]

/** Live result of the simulator's safety checker. */
export function InvariantBadge() {
  const violations = useSim((s) => s.state?.violations ?? NONE)
  const [open, setOpen] = useState(false)

  if (violations.length === 0) {
    return (
      <span
        className="flex items-center gap-1.5 rounded-full border border-leader/40 px-2.5 py-1 text-xs text-leader"
        title={`Checked continuously:\n${PROPERTIES.join('\n')}`}
        aria-label="Safety checks: all passing"
      >
        <ShieldCheck className="size-3.5" /> Safe
      </span>
    )
  }
  return (
    <div className="relative">
      <button
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
        className="flex items-center gap-1.5 rounded-full border border-destructive/50 bg-destructive/10 px-2.5 py-1 text-xs text-destructive"
      >
        <ShieldAlert className="size-3.5" /> {violations.length} safety violation
        {violations.length > 1 ? 's' : ''}
      </button>
      {open && (
        <ul
          aria-label="Violations"
          className="absolute right-0 z-10 mt-2 flex w-96 flex-col gap-2 rounded-xl border bg-popover p-3 text-sm shadow-lg"
        >
          {violations.map((v, i) => (
            <li key={i}>
              <p className="font-medium text-destructive">
                {v.invariant}{' '}
                <span className="font-mono text-xs text-muted-foreground">at {formatTime(v.time)}</span>
              </p>
              <p className="text-muted-foreground">{v.detail}</p>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
