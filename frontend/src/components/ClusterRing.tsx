import type { NodeID, NodeView } from '@/client/types'
import { RING, nodeColor, nodePositions, nodeStatus } from '@/lib/cluster'
import type { Point } from '@/lib/geometry'
import { cn } from '@/lib/utils'

interface Props {
  nodes: NodeView[]
  selected?: NodeID | null
  onSelect?: (id: NodeID) => void
  /** Extra layers drawn between links and nodes (e.g. message packets). */
  children?: React.ReactNode
}

export function ClusterRing({ nodes, selected, onSelect, children }: Props) {
  const pos = nodePositions(nodes.length)
  return (
    <svg viewBox={`0 0 ${RING.size} ${RING.size}`} className="h-full w-full select-none" aria-label="Cluster">
      <g className="stroke-border" strokeWidth={1.5}>
        {pos.flatMap((a, i) =>
          pos
            .slice(i + 1)
            .map((b, j) => <line key={`${i}-${i + j + 1}`} x1={a.x} y1={a.y} x2={b.x} y2={b.y} />),
        )}
      </g>
      {children}
      {nodes.map((n, i) => (
        <Node key={n.id} node={n} at={pos[i]} selected={selected === n.id} onSelect={onSelect} />
      ))}
    </svg>
  )
}

function Node({
  node: n,
  at,
  selected,
  onSelect,
}: {
  node: NodeView
  at: Point
  selected: boolean
  onSelect?: (id: NodeID) => void
}) {
  const color = nodeColor(n)
  const r = RING.node
  const timerR = r + 7
  const circumference = 2 * Math.PI * timerR
  const showTimer = n.state === 'up' && n.role !== 'leader'
  const progress = Math.min(n.electionElapsed / Math.max(n.electionTimeout, 1), 1)
  const label = `Node ${n.id}, ${nodeStatus(n)}, term ${n.term}`

  return (
    <g
      transform={`translate(${at.x} ${at.y})`}
      role="button"
      tabIndex={0}
      aria-label={label}
      aria-pressed={selected}
      className="cursor-pointer outline-none"
      onClick={() => onSelect?.(n.id)}
      onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && onSelect?.(n.id)}
    >
      {n.role === 'leader' && n.state === 'up' && (
        <circle r={r + 14} fill={color} opacity={0.12} className="animate-pulse" />
      )}
      {selected && (
        <circle r={r + 16} fill="none" stroke="var(--ring)" strokeWidth={2} strokeDasharray="4 4" />
      )}
      {showTimer && (
        <circle
          r={timerR}
          fill="none"
          stroke={color}
          strokeWidth={3}
          strokeLinecap="round"
          strokeDasharray={`${progress * circumference} ${circumference}`}
          transform="rotate(-90)"
          opacity={0.7}
          data-testid={`timer-${n.id}`}
        />
      )}
      <circle
        r={r}
        className={cn('fill-card', n.state === 'crashed' && 'opacity-60')}
        stroke={color}
        strokeWidth={n.role === 'leader' && n.state === 'up' ? 4 : 2.5}
        strokeDasharray={n.state === 'crashed' ? '6 5' : undefined}
      />
      <text y={-4} textAnchor="middle" className="fill-foreground text-[22px] font-semibold">
        N{n.id}
      </text>
      <text y={18} textAnchor="middle" className="fill-muted-foreground font-mono text-[12px]">
        T{n.term}
      </text>
      <text y={r + 26} textAnchor="middle" className="font-mono text-[12px] uppercase" fill={color}>
        {nodeStatus(n)}
      </text>
    </g>
  )
}
