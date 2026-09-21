import { Scissors, Unplug } from 'lucide-react'
import { NetworkConditions } from '@/components/NetworkConditions'
import { Button } from '@/components/ui/button'
import { useSim } from '@/store/sim'

/** Partition and heal the network. */
export function NetworkPanel() {
  const draft = useSim((s) => s.partitionDraft)
  const nodes = useSim((s) => s.state?.nodes ?? [])
  const anyCut = useSim((s) => s.state?.links.some((l) => l.cut) ?? false)
  const { startPartition, applyPartition, cancelPartition, act } = useSim((s) => s)

  const rest = nodes.map((n) => n.id).filter((id) => !draft?.includes(id))
  const names = (ids: number[]) => (ids.length ? ids.map((id) => `N${id}`).join(' ') : '—')

  return (
    <section aria-label="Network" className="flex flex-col gap-3 rounded-xl border bg-card p-4">
      <h2 className="font-semibold">Network</h2>
      {draft ? (
        <>
          <p className="text-sm text-muted-foreground">
            Click nodes to put them on one side of the partition.
          </p>
          <dl className="grid grid-cols-2 gap-2 font-mono text-sm">
            <div className="rounded-lg bg-primary/10 px-3 py-2">
              <dt className="text-xs text-muted-foreground">side A</dt>
              <dd>{names(draft)}</dd>
            </div>
            <div className="rounded-lg bg-secondary/60 px-3 py-2">
              <dt className="text-xs text-muted-foreground">side B</dt>
              <dd>{names(rest)}</dd>
            </div>
          </dl>
          <div className="grid grid-cols-2 gap-2">
            <Button onClick={applyPartition} disabled={!draft.length || !rest.length}>
              Apply
            </Button>
            <Button variant="outline" onClick={cancelPartition}>
              Cancel
            </Button>
          </div>
        </>
      ) : (
        <>
          <p className="text-sm text-muted-foreground">
            Click a link to cut it, or split the cluster in two.
          </p>
          <div className="grid grid-cols-2 gap-2">
            <Button variant="outline" onClick={startPartition}>
              <Scissors /> Partition…
            </Button>
            <Button variant="outline" onClick={() => act({ kind: 'heal' })} disabled={!anyCut}>
              <Unplug /> Heal all
            </Button>
          </div>
          <NetworkConditions />
        </>
      )}
    </section>
  )
}
