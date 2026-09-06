export type AgentLogStreamName = 'stdout' | 'stderr' | string

export interface AgentLogItem {
  id: string
  ts: string
  tsNs: string
  stream: AgentLogStreamName
  line: string
  truncated: boolean
}

export interface AgentLogsResponse {
  logs: AgentLogItem[]
  nextCursor: string
  previousCursor: string
  hasOlder: boolean
  hasNewer: boolean
  caughtUp: boolean
  gap: boolean
  gapReason: string
}
