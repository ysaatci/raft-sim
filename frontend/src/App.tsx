import { ClusterRing } from '@/components/ClusterRing'
import { ClusterSettings } from '@/components/ClusterSettings'
import { Failure } from '@/components/ErrorBoundary'
import { ErrorToast } from '@/components/ErrorToast'
import { EventFeed } from '@/components/EventFeed'
import { InvariantBadge } from '@/components/InvariantBadge'
import { Legend } from '@/components/Legend'
import { LogGrid } from '@/components/LogGrid'
import { Narration } from '@/components/Narration'
import { NetworkPanel } from '@/components/NetworkPanel'
import { NodePanel } from '@/components/NodePanel'
import { Packets } from '@/components/Packets'
import { PlaybackControls } from '@/components/PlaybackControls'
import { ScenarioPicker } from '@/components/ScenarioPicker'
import { Timeline } from '@/components/Timeline'
import { useSim } from '@/store/sim'
import { useShortcuts } from '@/store/useShortcuts'
import { useSimLoop } from '@/store/useSimLoop'

const SHORTCUTS = [
  ['Space', 'play / pause'],
  ['→', 'step'],
  ['1–9', 'select node'],
  ['Esc', 'deselect'],
]

export default function App() {
  useSimLoop()
  useShortcuts()
  const state = useSim((s) => s.state)
  const loadError = useSim((s) => s.loadError)
  const selected = useSim((s) => s.selected)
  const select = useSim((s) => s.select)
  const draft = useSim((s) => s.partitionDraft)
  const { togglePartitionNode, toggleLink } = useSim((s) => s)
  const node = state?.nodes.find((n) => n.id === selected)

  if (loadError) return <Failure title="The simulator could not start" message={loadError} />

  return (
    <div className="flex min-h-svh flex-col">
      <header className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3 sm:px-6">
        <div>
          <h1 className="text-lg font-semibold tracking-tight">Raft Simulator</h1>
          <p className="text-sm text-muted-foreground">Leader election and log replication, step by step</p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <InvariantBadge />
          <PlaybackControls />
        </div>
      </header>
      {state && <Timeline />}
      {state ? (
        <main className="grid flex-1 gap-6 p-4 sm:p-6 lg:grid-cols-[1fr_320px]">
          <div className="mx-auto flex w-full max-w-[640px] flex-col gap-2">
            <Narration />
            <div className="aspect-square w-full">
              <ClusterRing
                nodes={state.nodes}
                selected={selected}
                onSelect={(id) => (draft ? togglePartitionNode(id) : select(id === selected ? null : id))}
                links={state.links}
                onLinkClick={toggleLink}
                highlighted={draft ?? []}
              >
                <Packets flights={state.flights} nodes={state.nodes} time={state.time} />
              </ClusterRing>
            </div>
            <Legend />
          </div>
          <aside className="flex flex-col gap-4">
            <ScenarioPicker />
            {node ? (
              <NodePanel node={node} />
            ) : (
              <p className="rounded-xl border border-dashed p-4 text-sm text-muted-foreground">
                Click a node to crash, pause or restart it, or to send it a write.
              </p>
            )}
            <NetworkPanel />
            <EventFeed />
            <ClusterSettings key={JSON.stringify(state.config)} config={state.config} />
          </aside>
          <div className="lg:col-span-2">
            <LogGrid nodes={state.nodes} selected={selected} />
          </div>
        </main>
      ) : (
        <main className="grid flex-1 place-items-center">
          <p className="animate-pulse text-muted-foreground">Loading simulator…</p>
        </main>
      )}
      <footer className="hidden flex-wrap justify-center gap-x-4 gap-y-1 border-t px-6 py-2 text-xs text-muted-foreground sm:flex">
        {SHORTCUTS.map(([key, what]) => (
          <span key={key}>
            <kbd className="rounded border bg-secondary px-1 font-mono">{key}</kbd> {what}
          </span>
        ))}
      </footer>
      <ErrorToast />
    </div>
  )
}
