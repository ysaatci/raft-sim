import type { NodeView } from '@/client/types'
import { ringPositions, type Point } from './geometry'

export const RING = { size: 600, radius: 210, node: 44 }
const CENTER: Point = { x: RING.size / 2, y: RING.size / 2 }

/** Color for a node: crashed/paused override its Raft role. */
export function nodeColor(n: NodeView): string {
  if (n.state === 'crashed') return 'var(--crashed)'
  if (n.state === 'paused') return 'var(--paused)'
  return `var(--${n.role})`
}

/** What a node is doing, in one word: its role, or why it has none. */
export function nodeStatus(n: NodeView): string {
  return n.state === 'up' ? n.role : n.state
}

export function nodePositions(count: number): Point[] {
  return ringPositions(count, CENTER, RING.radius)
}

/** A distinct, stable color per term, so entries from different terms stand apart. */
export function termColor(term: number): string {
  return `oklch(0.7 0.13 ${(term * 67) % 360})`
}
