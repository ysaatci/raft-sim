import { Button } from '@/components/ui/button'

const roles = [
  ['Follower', 'bg-follower'],
  ['Candidate', 'bg-candidate'],
  ['Leader', 'bg-leader'],
  ['Crashed', 'bg-crashed'],
  ['Paused', 'bg-paused'],
] as const

export default function App() {
  return (
    <main className="mx-auto flex min-h-svh max-w-3xl flex-col items-start gap-6 p-10">
      <h1 className="text-3xl font-semibold tracking-tight">Raft Simulator</h1>
      <ul className="flex flex-wrap gap-4 font-mono text-sm text-muted-foreground">
        {roles.map(([name, bg]) => (
          <li key={name} className="flex items-center gap-2">
            <span className={`size-3 rounded-full ${bg}`} />
            {name}
          </li>
        ))}
      </ul>
      <Button>Start</Button>
    </main>
  )
}
