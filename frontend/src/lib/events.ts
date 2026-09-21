import type { SimEvent } from '@/client/types'

export type EventCategory = 'election' | 'log' | 'fault'

export const CATEGORY: Record<SimEvent['type'], EventCategory> = {
  'election-started': 'election',
  'became-leader': 'election',
  'stepped-down': 'election',
  'vote-granted': 'election',
  'log-truncated': 'log',
  'commit-advanced': 'log',
  'node-crashed': 'fault',
  'node-restarted': 'fault',
  'node-paused': 'fault',
  'node-resumed': 'fault',
}

/** One event as a sentence. */
export function describe(e: SimEvent): string {
  const n = `N${e.node}`
  switch (e.type) {
    case 'election-started':
      return `${n} started an election for term ${e.term}`
    case 'became-leader':
      return `${n} became leader of term ${e.term}`
    case 'stepped-down':
      return `${n} stepped down (term ${e.term})`
    case 'vote-granted':
      return `${n} voted for N${e.peer} in term ${e.term}`
    case 'log-truncated':
      return `${n} discarded conflicting entries from #${e.index}`
    case 'commit-advanced':
      return `${n} committed up to #${e.index}`
    case 'node-crashed':
      return `${n} crashed`
    case 'node-restarted':
      return `${n} restarted`
    case 'node-paused':
      return `${n} paused`
    case 'node-resumed':
      return `${n} resumed`
  }
}

export function formatTime(ms: number): string {
  return `${(ms / 1000).toFixed(3)}s`
}
