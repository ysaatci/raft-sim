import type { LinkView, NodeID, NodeView } from '@/client/types'
import { RING, linkStatus, nodeColor, nodePositions, nodeStatus, type LinkStatus } from '@/lib/cluster'
import type { Point } from '@/lib/geometry'
import { cn } from '@/lib/utils'

interface Props {
  nodes: NodeView[]
  selected?: NodeID | null
  onSelect?: (id: NodeID) => void
  links?: LinkView[]
  onLinkClick?: (a: NodeID, b: NodeID) => void
  /** Nodes to mark, e.g. one side of a partition being drawn. */
  highlighted?: NodeID[]
  /** Extra layers drawn between links and nodes (e.g. message packets). */
  children?: React.ReactNode
}

export function ClusterRing({
  nodes,
  selected,
  onSelect,
  links = [],
  onLinkClick,
  highlighted = [],
  children,
}: Props) {
  const pos = nodePositions(nodes.length)
  const pairs = nodes.flatMap((a, i) => nodes.slice(i + 1).map((b, j) => [a.id, b.id, i, i + j + 1] as const))
  return (
    <svg viewBox={`0 0 ${RING.size} ${RING.size}`} className="h-full w-full select-none" aria-label="Cluster">
      <g aria-label="Links">
        {pairs.map(([a, b, i, j]) => (
          <Link
            key={`${a}-${b}`}
            a={a}
            b={b}
            from={pos[i]}
            to={pos[j]}
            status={linkStatus(links, a, b)}
            onClick={onLinkClick}
          />
        ))}
      </g>
      {children}
      {nodes.map((n, i) => (
        <Node
          key={n.id}
          node={n}
          at={pos[i]}
          selected={selected === n.id}
          highlighted={highlighted.includes(n.id)}
          onSelect={onSelect}
        />
      ))}
    </svg>
  )
}

function Node({
  node: n,
  at,
  selected,
  highlighted,
  onSelect,
}: {
  node: NodeView
  at: Point
  selected: boolean
  highlighted: boolean
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
      {highlighted && (
        <circle r={r + 12} fill="var(--primary)" fillOpacity={0.12} stroke="var(--primary)" strokeWidth={2} />
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

const LINK_STYLE: Record<LinkStatus, { stroke: string; dash?: string; label: string }> = {
  up: { stroke: 'var(--border)', label: 'connected' },
  cut: { stroke: 'var(--crashed)', dash: '6 6', label: 'cut' },
  'one-way': { stroke: 'var(--candidate)', dash: '2 6', label: 'cut in one direction' },
}

function Link({
  a,
  b,
  from,
  to,
  status,
  onClick,
}: {
  a: NodeID
  b: NodeID
  from: Point
  to: Point
  status: LinkStatus
  onClick?: (a: NodeID, b: NodeID) => void
}) {
  const style = LINK_STYLE[status]
  const line = { x1: from.x, y1: from.y, x2: to.x, y2: to.y }
  const interactive = onClick
    ? {
        role: 'button',
        tabIndex: 0,
        'aria-label': `Link N${a}–N${b}, ${style.label}`,
        className: 'cursor-pointer outline-none [&:hover>line:first-child]:opacity-100',
        onClick: () => onClick(a, b),
        onKeyDown: (e: React.KeyboardEvent) => (e.key === 'Enter' || e.key === ' ') && onClick(a, b),
      }
    : {}
  return (
    <g {...interactive} data-status={status}>
      <line
        {...line}
        stroke={style.stroke}
        strokeWidth={status === 'up' ? 1.5 : 2.5}
        strokeDasharray={style.dash}
      />
      {onClick && <line {...line} stroke="transparent" strokeWidth={16} />}
    </g>
  )
}
