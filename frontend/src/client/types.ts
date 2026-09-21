// Mirrors the JSON produced by backend/sim (State, Action, TimedEvent).

export type NodeID = number
export type Role = 'follower' | 'candidate' | 'leader'
export type NodeState = 'up' | 'crashed' | 'paused'
export type MsgType = 'RequestVote' | 'RequestVoteResp' | 'AppendEntries' | 'AppendEntriesResp'

export interface Entry {
  term: number
  index: number
  data: string
}

export interface Message {
  type: MsgType
  from: NodeID
  to: NodeID
  term: number
  lastLogIndex?: number
  lastLogTerm?: number
  prevLogIndex?: number
  prevLogTerm?: number
  entries?: Entry[]
  commit?: number
  reject?: boolean
  matchIndex?: number
  conflictIndex?: number
  conflictTerm?: number
}

export interface LinkConfig {
  latencyMs: number
  jitterMs: number
  dropRate: number
  cut: boolean
}

export interface Config {
  size: number
  seed: number
  tickMs: number
  electionTick: number
  heartbeatTick: number
  network: LinkConfig
}

export interface NodeView {
  id: NodeID
  role: Role
  term: number
  votedFor: NodeID
  leader: NodeID
  commit: number
  applied: number
  lastIndex: number
  lastTerm: number
  electionElapsed: number
  electionTimeout: number
  state: NodeState
  log: Entry[]
  next?: Record<string, number>
  match?: Record<string, number>
  kv: Record<string, string>
  inbox: number
}

export interface Flight {
  id: number
  msg: Message
  sentAt: number
  deliverAt: number
  dropped?: boolean
}

export interface LinkView extends LinkConfig {
  from: NodeID
  to: NodeID
}

export interface Violation {
  time: number
  invariant: string
  detail: string
}

export interface SimState {
  time: number
  config: Config
  /** Current conditions of links without their own settings. */
  network: LinkConfig
  nodes: NodeView[]
  flights: Flight[]
  links: LinkView[]
  violations: Violation[]
}

export type EventType =
  | 'election-started'
  | 'became-leader'
  | 'stepped-down'
  | 'vote-granted'
  | 'log-truncated'
  | 'commit-advanced'
  | 'node-crashed'
  | 'node-restarted'
  | 'node-paused'
  | 'node-resumed'

export interface SimEvent {
  time: number
  type: EventType
  node: NodeID
  term: number
  peer?: NodeID
  index?: number
}

export type Action =
  | { kind: 'crash' | 'restart' | 'pause' | 'resume'; node: NodeID }
  | { kind: 'propose'; node: NodeID; data: string }
  | { kind: 'set-link'; node: NodeID; to: NodeID; link: LinkConfig }
  | { kind: 'set-network'; link: LinkConfig }
  | { kind: 'partition'; groups: NodeID[][] }
  | { kind: 'heal' }
