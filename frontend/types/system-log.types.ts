export type SystemLogStreamName = "stdout" | "stderr" | string

export interface SystemLogItem {
  id: string
  ts: string
  tsNs: string
  stream: SystemLogStreamName
  line: string
  truncated: boolean
}

export interface SystemLogsResponse {
  logs: SystemLogItem[]
  nextCursor: string
  previousCursor: string
  hasOlder: boolean
  hasNewer: boolean
  caughtUp: boolean
  gap: boolean
  gapReason: string
}
