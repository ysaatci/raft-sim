import { useEffect, useRef } from 'react'
import type { NodeID, NodeView } from '@/client/types'
import { nodeColor, termColor } from '@/lib/cluster'
import { cn } from '@/lib/utils'

/**
 * Every node's log side by side: one row per node, one column per index.
 * Solid cells are committed on that node, outlined ones are not yet.
 */
export function LogGrid({ nodes, selected }: { nodes: NodeView[]; selected?: NodeID | null }) {
  const width = Math.max(...nodes.map((n) => n.lastIndex), 0)
  const scroller = useRef<HTMLDivElement>(null)

  // Follow the end of the log as it grows, unless the user scrolled back.
  const pinned = useRef(true)
  useEffect(() => {
    const el = scroller.current
    if (el && pinned.current) el.scrollLeft = el.scrollWidth
  }, [width])

  return (
    <section aria-label="Logs" className="rounded-xl border bg-card p-4">
      <header className="mb-3 flex items-baseline justify-between">
        <h2 className="font-semibold">Logs</h2>
        <p className="text-xs text-muted-foreground">
          color = term · solid = committed · hover for the command
        </p>
      </header>
      <div
        ref={scroller}
        className="overflow-x-auto pb-2"
        onScroll={(e) => {
          const el = e.currentTarget
          pinned.current = el.scrollLeft + el.clientWidth >= el.scrollWidth - 4
        }}
      >
        <table className="border-separate border-spacing-1 font-mono text-[11px]">
          <thead>
            <tr>
              <th />
              {Array.from({ length: width }, (_, i) => (
                <th key={i} scope="col" className="w-7 font-normal text-muted-foreground">
                  {i + 1}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {nodes.map((n) => (
              <tr
                key={n.id}
                aria-label={`Node ${n.id} log`}
                className={cn(n.state === 'crashed' && 'opacity-50')}
              >
                <th
                  scope="row"
                  className={cn('pr-2 text-left font-semibold', selected === n.id && 'underline')}
                  style={{ color: nodeColor(n) }}
                >
                  N{n.id}
                </th>
                {Array.from({ length: width }, (_, i) => {
                  const e = n.log[i]
                  if (!e) return <td key={i} className="h-7 w-7" />
                  const committed = e.index <= n.commit
                  const color = termColor(e.term)
                  return (
                    <td
                      key={i}
                      title={`#${e.index} term ${e.term}: ${e.data || '(no-op)'}${committed ? '' : ' (uncommitted)'}`}
                      data-committed={committed}
                      className="h-7 w-7 rounded-md border-2 text-center font-semibold"
                      style={{
                        borderColor: color,
                        background: committed ? color : 'transparent',
                        color: committed ? 'var(--background)' : color,
                      }}
                    >
                      {e.term}
                    </td>
                  )
                })}
              </tr>
            ))}
          </tbody>
        </table>
        {width === 0 && (
          <p className="text-sm text-muted-foreground">No entries yet. Elect a leader and send a write.</p>
        )}
      </div>
    </section>
  )
}
