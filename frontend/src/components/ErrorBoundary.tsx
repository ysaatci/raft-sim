import { Component, type ReactNode } from 'react'
import { Button } from '@/components/ui/button'

/** Shows what went wrong instead of a blank page. */
export class ErrorBoundary extends Component<{ children: ReactNode }, { error: Error | null }> {
  state = { error: null as Error | null }

  static getDerivedStateFromError(error: Error) {
    return { error }
  }

  render() {
    if (!this.state.error) return this.props.children
    return <Failure title="Something went wrong" message={this.state.error.message} />
  }
}

export function Failure({ title, message }: { title: string; message: string }) {
  return (
    <main role="alert" className="grid min-h-svh place-items-center p-6">
      <div className="flex max-w-md flex-col items-center gap-3 text-center">
        <h1 className="text-lg font-semibold">{title}</h1>
        <p className="font-mono text-sm text-muted-foreground">{message}</p>
        <Button onClick={() => location.reload()}>Reload</Button>
      </div>
    </main>
  )
}
