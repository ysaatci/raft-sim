import { useState } from 'react'
import type { LinkConfig, NodeID } from '@/client/types'
import { Slider } from '@/components/ui/slider'
import { useSim } from '@/store/sim'

type Conditions = Omit<LinkConfig, 'cut'>

const SLIDERS: {
  key: keyof Conditions
  label: string
  max: number
  step: number
  format: (v: number) => string
}[] = [
  { key: 'latencyMs', label: 'Latency', max: 200, step: 1, format: (v) => `${v} ms` },
  { key: 'jitterMs', label: 'Jitter', max: 100, step: 1, format: (v) => `±${v} ms` },
  { key: 'dropRate', label: 'Packet loss', max: 0.5, step: 0.01, format: (v) => `${Math.round(v * 100)}%` },
]

/** Latency, jitter and loss for every link, or for one link in both directions. */
export function NetworkConditions() {
  const state = useSim((s) => s.state)
  const act = useSim((s) => s.act)
  const [scope, setScope] = useState('all') // 'all' or 'a-b'
  const [draft, setDraft] = useState<Partial<Conditions>>({}) // values while dragging

  if (!state) return null
  const ids = state.nodes.map((n) => n.id)
  const pairs = ids.flatMap((a, i) => ids.slice(i + 1).map((b) => [a, b] as const))
  const [a, b] = scope === 'all' ? [0, 0] : scope.split('-').map(Number)
  const current: LinkConfig =
    scope === 'all' ? state.network : state.links.find((l) => l.from === a && l.to === b)!

  const commit = async (patch: Partial<Conditions>) => {
    setDraft({})
    if (scope === 'all') {
      await act({ kind: 'set-network', link: { ...state.network, ...patch, cut: false } })
      return
    }
    for (const [from, to] of [
      [a, b],
      [b, a],
    ] as [NodeID, NodeID][]) {
      const l = state.links.find((l) => l.from === from && l.to === to)!
      const { latencyMs, jitterMs, dropRate, cut } = l
      await act({ kind: 'set-link', node: from, to, link: { latencyMs, jitterMs, dropRate, cut, ...patch } })
    }
  }

  return (
    <div className="flex flex-col gap-3 border-t pt-3">
      <label className="flex items-center justify-between gap-2 text-sm">
        <span className="text-muted-foreground">Conditions for</span>
        <select
          value={scope}
          onChange={(e) => {
            setScope(e.target.value)
            setDraft({})
          }}
          className="rounded-md border bg-background px-2 py-1 font-mono text-xs"
        >
          <option value="all">all links</option>
          {pairs.map(([x, y]) => (
            <option key={`${x}-${y}`} value={`${x}-${y}`}>
              N{x} ↔ N{y}
            </option>
          ))}
        </select>
      </label>
      {SLIDERS.map((s) => {
        const value = draft[s.key] ?? current[s.key]
        return (
          <div key={s.key} className="flex flex-col gap-1.5">
            <div className="flex justify-between text-xs">
              <span className="text-muted-foreground">{s.label}</span>
              <span className="font-mono tabular-nums">{s.format(value)}</span>
            </div>
            <Slider
              aria-label={s.label}
              min={0}
              max={s.max}
              step={s.step}
              value={[value]}
              onValueChange={([v]) => setDraft((d) => ({ ...d, [s.key]: v }))}
              onValueCommit={([v]) => commit({ [s.key]: v })}
            />
          </div>
        )
      })}
    </div>
  )
}
