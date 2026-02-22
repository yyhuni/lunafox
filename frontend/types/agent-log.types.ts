export type AgentLogStreamName = 'stdout' | 'stderr' | string

export interface AgentLogStatusEvent {
  type: 'status'
  requestId: string
  status: string
}

export interface AgentLogChunkEvent {
  type: 'log'
  requestId: string
  ts: string
  stream: AgentLogStreamName
  line: string
  truncated: boolean
}

export interface AgentLogErrorEvent {
  type: 'error'
  requestId: string
  code: string
  message: string
}

export interface AgentLogDoneEvent {
  type: 'done'
  requestId: string
  reason: string
}

export interface AgentLogPingEvent {
  type: 'ping'
  requestId: string
  ts: string
}

export type AgentLogStreamEvent =
  | AgentLogStatusEvent
  | AgentLogChunkEvent
  | AgentLogErrorEvent
  | AgentLogDoneEvent
  | AgentLogPingEvent
