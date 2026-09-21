import { ClusterRing } from '@/components/ClusterRing'
import { ErrorToast } from '@/components/ErrorToast'
import { NodePanel } from '@/components/NodePanel'
import { PlaybackControls } from '@/components/PlaybackControls'
import { useSim } from '@/store/sim'
import { useSimLoop } from '@/store/useSimLoop'

export default function App() {
  useSimLoop()
  const state = useSim((s) => s.state)
  const selected = useSim((s) => s.selected)
  const select = useSim((s) => s.select)
  const node = state?.nodes.find((n) => n.id === selected)

  return (
    <div className="flex min-h-svh flex-col">
      <header className="flex flex-wrap items-center justify-between gap-4 border-b px-6 py-3">
        <div>
          <h1 className="text-lg font-semibold tracking-tight">Raft Simulator</h1>
          <p className="text-sm text-muted-foreground">Leader election and log replication, step by step</p>
        </div>
        <PlaybackControls />
      </header>
      {state ? (
        <main className="grid flex-1 gap-6 p-6 lg:grid-cols-[1fr_320px]">
          <div className="mx-auto aspect-square w-full max-w-[640px]">
            <ClusterRing
              nodes={state.nodes}
              selected={selected}
              onSelect={(id) => select(id === selected ? null : id)}
            />
          </div>
          <aside className="flex flex-col gap-4">
            {node ? (
              <NodePanel node={node} />
            ) : (
              <p className="rounded-xl border border-dashed p-4 text-sm text-muted-foreground">
                Click a node to crash, pause or restart it, or to send it a write.
              </p>
            )}
          </aside>
        </main>
      ) : (
        <main className="grid flex-1 place-items-center">
          <p className="text-muted-foreground">Loading simulator…</p>
        </main>
      )}
      <ErrorToast />
    </div>
  )
}
