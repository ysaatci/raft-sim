import type { Method, RawFrame } from './engine'

export interface Request {
  id: number
  method: Method
  arg: string | number
}

export type Response = { id: number; frame: RawFrame } | { id: number; error: string }
