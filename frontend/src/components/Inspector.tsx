import type { NodeView } from '@/client/types'

/** The Raft state of one node, in detail. Rendered inside NodePanel. */
export function Inspector({ node: n }: { node: NodeView }) {
  const kv = Object.entries(n.kv).sort(([a], [b]) => a.localeCompare(b))
  const peers = Object.keys(n.match ?? {})
    .map(Number)
    .filter((id) => id !== n.id)

  return (
    <div className="flex flex-col gap-4 text-sm">
      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 font-mono">
        <Row label="voted for" value={n.votedFor ? `N${n.votedFor}` : '—'} />
        <Row label="leader" value={n.leader ? `N${n.leader}` : 'unknown'} />
        <Row label="last log" value={n.lastIndex ? `#${n.lastIndex} (term ${n.lastTerm})` : 'empty'} />
        <Row label="applied" value={n.applied} />
        {n.state === 'up' && n.role !== 'leader' && (
          <Row label="election timer" value={`${n.electionElapsed} / ${n.electionTimeout} ticks`} />
        )}
        {n.state === 'paused' && <Row label="queued" value={`${n.inbox} messages`} />}
      </dl>

      {n.role === 'leader' && n.state === 'up' && peers.length > 0 && (
        <div>
          <h3 className="mb-1 text-xs font-medium text-muted-foreground">Replication (match / next)</h3>
          <ul className="flex flex-col gap-1.5" aria-label="Replication">
            {peers.map((id) => {
              const match = n.match![id]
              const next = n.next![id]
              const pct = n.lastIndex ? (match / n.lastIndex) * 100 : 100
              return (
                <li
                  key={id}
                  className="grid grid-cols-[2rem_1fr_4.5rem] items-center gap-2 font-mono text-xs"
                >
                  <span>N{id}</span>
                  <span className="h-1.5 overflow-hidden rounded-full bg-secondary">
                    <span className="block h-full rounded-full bg-leader" style={{ width: `${pct}%` }} />
                  </span>
                  <span
                    className="text-right tabular-nums text-muted-foreground"
                    aria-label={`N${id} progress`}
                  >
                    {match} / {next}
                  </span>
                </li>
              )
            })}
          </ul>
        </div>
      )}

      <div>
        <h3 className="mb-1 text-xs font-medium text-muted-foreground">State machine</h3>
        {kv.length ? (
          <table className="w-full font-mono text-xs" aria-label="State machine">
            <tbody>
              {kv.map(([k, v]) => (
                <tr key={k} className="border-b last:border-0">
                  <td className="py-1 text-muted-foreground">{k}</td>
                  <td className="py-1 text-right">{v}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="text-xs text-muted-foreground">empty</p>
        )}
      </div>
    </div>
  )
}

function Row({ label, value }: { label: string; value: string | number }) {
  return (
    <>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="text-right tabular-nums">{value}</dd>
    </>
  )
}
