import { ClusterRing } from '@/components/ClusterRing'
import { PlaybackControls } from '@/components/PlaybackControls'
import { useSim } from '@/store/sim'
import { useSimLoop } from '@/store/useSimLoop'

export default function App() {
  useSimLoop()
  const state = useSim((s) => s.state)

  return (
    <div className="flex min-h-svh flex-col">
      <header className="flex flex-wrap items-center justify-between gap-4 border-b px-6 py-3">
        <div>
          <h1 className="text-lg font-semibold tracking-tight">Raft Simulator</h1>
          <p className="text-sm text-muted-foreground">Leader election and log replication, step by step</p>
        </div>
        <PlaybackControls />
      </header>
      <main className="grid flex-1 place-items-center p-6">
        {state ? (
          <div className="aspect-square w-full max-w-[640px]">
            <ClusterRing nodes={state.nodes} />
          </div>
        ) : (
          <p className="text-muted-foreground">Loading simulator…</p>
        )}
      </main>
    </div>
  )
}
