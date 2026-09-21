import { BookOpen } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useSim } from '@/store/sim'

/** The built-in guided scenarios. */
export function ScenarioPicker() {
  const scenarios = useSim((s) => s.scenarios)
  const current = useSim((s) => s.scenario?.id)
  const loadScenario = useSim((s) => s.loadScenario)
  if (scenarios.length === 0) return null

  return (
    <details open className="rounded-xl border bg-card p-4" aria-label="Guided scenarios">
      <summary className="flex cursor-pointer items-center gap-2 font-semibold marker:text-muted-foreground">
        <BookOpen className="size-4" /> Guided scenarios
      </summary>
      <ul className="mt-3 flex flex-col gap-1">
        {scenarios.map((sc) => (
          <li key={sc.id}>
            <button
              onClick={() => loadScenario(sc.id)}
              aria-current={current === sc.id}
              className={cn(
                'w-full rounded-lg px-3 py-2 text-left transition-colors hover:bg-secondary/60',
                current === sc.id && 'bg-secondary',
              )}
            >
              <span className="block text-sm font-medium">{sc.title}</span>
              <span className="block text-xs text-muted-foreground">{sc.summary}</span>
            </button>
          </li>
        ))}
      </ul>
    </details>
  )
}
