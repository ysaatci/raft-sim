import { Pause, Play, Power, PowerOff, Send, X } from 'lucide-react'
import { useState } from 'react'
import type { NodeView } from '@/client/types'
import { Button } from '@/components/ui/button'
import { nodeColor, nodeStatus } from '@/lib/cluster'
import { useSim } from '@/store/sim'

const KEYS = ['x', 'y', 'z']

/** Side panel for the selected node: its state and what you can do to it. */
export function NodePanel({ node: n }: { node: NodeView }) {
  const act = useSim((s) => s.act)
  const select = useSim((s) => s.select)
  const [writes, setWrites] = useState(0)
  const command = `set ${KEYS[writes % KEYS.length]} ${writes + 1}`

  const send = async () => {
    if (await act({ kind: 'propose', node: n.id, data: command })) setWrites((w) => w + 1)
  }

  return (
    <section aria-label={`Node ${n.id}`} className="flex flex-col gap-4 rounded-xl border bg-card p-4">
      <header className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="size-3 rounded-full" style={{ background: nodeColor(n) }} />
          <h2 className="font-semibold">Node {n.id}</h2>
          <span className="font-mono text-xs uppercase" style={{ color: nodeColor(n) }}>
            {nodeStatus(n)}
          </span>
        </div>
        <Button size="icon-sm" variant="ghost" onClick={() => select(null)} aria-label="Close">
          <X />
        </Button>
      </header>

      <dl className="grid grid-cols-3 gap-2 font-mono text-sm">
        <Stat label="term" value={n.term} />
        <Stat label="log" value={n.lastIndex} />
        <Stat label="commit" value={n.commit} />
      </dl>

      <div className="grid grid-cols-2 gap-2">
        {n.state === 'crashed' ? (
          <Button variant="outline" onClick={() => act({ kind: 'restart', node: n.id })}>
            <Power /> Restart
          </Button>
        ) : (
          <Button variant="destructive" onClick={() => act({ kind: 'crash', node: n.id })}>
            <PowerOff /> Crash
          </Button>
        )}
        {n.state === 'paused' ? (
          <Button variant="outline" onClick={() => act({ kind: 'resume', node: n.id })}>
            <Play /> Resume
          </Button>
        ) : (
          <Button
            variant="outline"
            disabled={n.state === 'crashed'}
            onClick={() => act({ kind: 'pause', node: n.id })}
          >
            <Pause /> Pause
          </Button>
        )}
      </div>
      <Button variant="secondary" disabled={n.state !== 'up'} onClick={send}>
        <Send /> Send write <code className="font-mono text-xs text-muted-foreground">{command}</code>
      </Button>
    </section>
  )
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg bg-secondary/60 px-3 py-2">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="text-lg tabular-nums">{value}</dd>
    </div>
  )
}
