export interface Point {
  x: number
  y: number
}

/** Positions for `count` nodes evenly spaced on a circle, node 1 at the top. */
export function ringPositions(count: number, center: Point, radius: number): Point[] {
  return Array.from({ length: count }, (_, i) => {
    const angle = -Math.PI / 2 + (i * 2 * Math.PI) / count
    return { x: center.x + radius * Math.cos(angle), y: center.y + radius * Math.sin(angle) }
  })
}
