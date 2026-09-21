import type { Method, RawFrame } from './engine'

/** Queries that return plain JSON rather than a frame. */
export type DataMethod = 'scenarios' | 'actions'

export type Request =
  { id: number; method: Method; arg: string | number } | { id: number; method: DataMethod }

export type Response =
  { id: number; frame: RawFrame } | { id: number; data: string } | { id: number; error: string }
