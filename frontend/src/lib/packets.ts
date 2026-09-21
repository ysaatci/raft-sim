import type { Flight } from '@/client/types'
import type { Point } from './geometry'

export interface PacketPos extends Point {
  opacity: number
}

/** Offset between the two directions of one link, so they don't overlap. */
const LANE = 7

/**
 * Where a message is drawn at virtual time `now`: interpolated between the
 * edges of the sender's and receiver's circles. Dropped messages fade out
 * and vanish halfway. Returns null if the packet should not be drawn.
 */
export function packetPosition(
  f: Flight,
  now: number,
  from: Point,
  to: Point,
  nodeRadius: number,
): PacketPos | null {
  const span = Math.max(f.deliverAt - f.sentAt, 1)
  const t = (now - f.sentAt) / span
  if (t < 0 || t > 1) return null
  if (f.dropped && t > 0.5) return null

  const dx = to.x - from.x
  const dy = to.y - from.y
  const len = Math.hypot(dx, dy) || 1
  const ux = dx / len
  const uy = dy / len
  // Start and end on the circles' edges, on the right-hand lane.
  const start = { x: from.x + ux * nodeRadius - uy * LANE, y: from.y + uy * nodeRadius + ux * LANE }
  const travel = len - 2 * nodeRadius
  return {
    x: start.x + ux * travel * t,
    y: start.y + uy * travel * t,
    opacity: f.dropped ? 1 - 2 * t : 1,
  }
}
