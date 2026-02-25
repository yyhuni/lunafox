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
  hasMore: boolean
}
