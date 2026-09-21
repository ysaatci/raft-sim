import type { Method, RawFrame } from './engine'

export type Request =
  { id: number; method: Method; arg: string | number } | { id: number; method: 'scenarios' }

export type Response =
  { id: number; frame: RawFrame } | { id: number; data: string } | { id: number; error: string }
