import { Dices } from 'lucide-react'
import { useState } from 'react'
import type { Config } from '@/client/types'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { useSim } from '@/store/sim'

const SIZES = [3, 5, 7]

/** Cluster size, seed and Raft timings. Applying restarts the simulation. */
export function ClusterSettings({ config }: { config: Config }) {
  const reset = useSim((s) => s.reset)
  const [size, setSize] = useState(config.size)
  // Inputs keep the text as typed, so a field can be cleared and retyped.
  const [seedText, setSeedText] = useState(String(config.seed))
  const [heartbeatText, setHeartbeatText] = useState(String(config.heartbeatTick * config.tickMs))
  const [electionText, setElectionText] = useState(String(config.electionTick * config.tickMs))
  const seed = Number(seedText)
  const heartbeatMs = Number(heartbeatText)
  const electionMs = Number(electionText)

  const invalid =
    !(seed >= 1) || !(heartbeatMs >= config.tickMs)
      ? `Seed must be at least 1 and the heartbeat at least ${config.tickMs} ms.`
      : electionMs <= heartbeatMs
        ? 'The election timeout must be longer than the heartbeat interval.'
        : null

  const apply = () =>
    reset({
      ...config,
      size,
      seed,
      heartbeatTick: Math.max(1, Math.round(heartbeatMs / config.tickMs)),
      electionTick: Math.max(2, Math.round(electionMs / config.tickMs)),
    })

  return (
    <details className="group rounded-xl border bg-card p-4" aria-label="Cluster settings">
      <summary className="cursor-pointer font-semibold marker:text-muted-foreground">Cluster</summary>
      <div className="mt-3 flex flex-col gap-3 text-sm">
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground">Nodes</span>
          <div className="flex rounded-lg border p-0.5" role="group" aria-label="Nodes">
            {SIZES.map((n) => (
              <button
                key={n}
                aria-pressed={size === n}
                onClick={() => setSize(n)}
                className={cn('rounded-md px-2.5 py-0.5 font-mono', size === n && 'bg-secondary')}
              >
                {n}
              </button>
            ))}
          </div>
        </div>
        <label className="flex items-center justify-between gap-2">
          <span className="text-muted-foreground">Seed</span>
          <span className="flex gap-1">
            <input
              type="number"
              min={1}
              value={seedText}
              onChange={(e) => setSeedText(e.target.value)}
              className="w-24 rounded-md border bg-background px-2 py-1 text-right font-mono"
            />
            <Button
              size="icon-sm"
              variant="outline"
              aria-label="Random seed"
              onClick={() => setSeedText(String(1 + Math.floor(Math.random() * 1e6)))}
            >
              <Dices />
            </Button>
          </span>
        </label>
        <NumberField
          label="Heartbeat every"
          value={heartbeatText}
          onChange={setHeartbeatText}
          step={config.tickMs}
        />
        <NumberField
          label="Election timeout ≥"
          value={electionText}
          onChange={setElectionText}
          step={config.tickMs}
        />
        <p className="text-xs text-muted-foreground">
          Each node waits a random time between the election timeout and twice that before starting an
          election.
        </p>
        {invalid && (
          <p role="alert" className="text-xs text-destructive">
            {invalid}
          </p>
        )}
        <Button onClick={apply} disabled={!!invalid}>
          Apply and restart
        </Button>
      </div>
    </details>
  )
}

function NumberField({
  label,
  value,
  onChange,
  step,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  step: number
}) {
  return (
    <label className="flex items-center justify-between gap-2">
      <span className="text-muted-foreground">{label}</span>
      <span className="flex items-center gap-1 font-mono">
        <input
          type="number"
          min={step}
          step={step}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="w-24 rounded-md border bg-background px-2 py-1 text-right"
        />
        ms
      </span>
    </label>
  )
}
