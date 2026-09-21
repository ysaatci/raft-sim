import type { Flight, MsgType, NodeView } from '@/client/types'
import { RING, nodePositions } from '@/lib/cluster'
import { packetPosition } from '@/lib/packets'

const COLOR: Record<MsgType, string> = {
  RequestVote: 'var(--msg-vote)',
  RequestVoteResp: 'var(--msg-vote)',
  AppendEntries: 'var(--msg-append)',
  AppendEntriesResp: 'var(--msg-append)',
}

/** In-flight messages, drawn as an SVG layer inside ClusterRing. */
export function Packets({ flights, nodes, time }: { flights: Flight[]; nodes: NodeView[]; time: number }) {
  const pos = nodePositions(nodes.length)
  return (
    <g aria-label="Messages">
      {flights.map((f) => {
        const p = packetPosition(f, time, pos[f.msg.from - 1], pos[f.msg.to - 1], RING.node)
        if (!p) return null
        const color = COLOR[f.msg.type]
        const isResponse = f.msg.type.endsWith('Resp')
        const entries = f.msg.entries?.length ?? 0
        const r = entries > 0 ? 8 : isResponse ? 4.5 : 5
        return (
          <g
            key={f.id}
            transform={`translate(${p.x} ${p.y})`}
            opacity={p.opacity}
            data-testid="packet"
            data-type={f.msg.type}
            data-dropped={f.dropped || undefined}
          >
            <title>
              {f.msg.type} {f.msg.from}→{f.msg.to} (term {f.msg.term})
              {entries > 0 ? `, ${entries} entries` : ''}
              {f.msg.reject ? ', rejected' : ''}
            </title>
            <circle
              r={r}
              fill={isResponse ? 'var(--background)' : color}
              stroke={f.msg.reject ? 'var(--crashed)' : color}
              strokeWidth={isResponse ? 2 : 0}
            />
            {entries > 0 && (
              <text textAnchor="middle" dy={3.5} className="fill-background font-mono text-[9px] font-bold">
                {entries}
              </text>
            )}
          </g>
        )
      })}
    </g>
  )
}
