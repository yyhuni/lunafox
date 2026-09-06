import type { SystemLogItem } from "@/types/system-log.types"

/**
 * Transport shape returned by GET /v1/admin/system/logEntries.
 * Keep this separate from SystemLogsResponse, which is the normalized client
 * result returned by systemLogService after it maps AIP pagination fields.
 */
export interface MockSystemLogEntriesResponse {
  results: SystemLogItem[]
  nextPageToken: string
  previousPageToken: string
  hasOlder: boolean
  hasNewer: boolean
  caughtUp: boolean
  gap: boolean
  gapReason: string
}

const MOCK_SYSTEM_LOG_HISTORY_SIZE = 5000
const MOCK_SYSTEM_LOG_START_MS = Date.parse("2024-12-28T08:36:46Z")

const mockSystemLogTailItems: SystemLogItem[] = [
  {
    id: "srv:lunafox-server:1735370400000000000:stdout:1f3b7f3a:000000",
    ts: "2024-12-28T10:00:00Z",
    tsNs: "1735370400000000000",
    stream: "stdout",
    line: "{\"level\":\"info\",\"timestamp\":\"2024-12-28T10:00:00Z\",\"caller\":\"server/main.go:42\",\"msg\":\"server started\",\"http.server.port\":8080}",
    truncated: false,
  },
  {
    id: "srv:lunafox-server:1735370401000000000:stdout:2b4c8d1e:000000",
    ts: "2024-12-28T10:00:01Z",
    tsNs: "1735370401000000000",
    stream: "stdout",
    line: "{\"level\":\"info\",\"timestamp\":\"2024-12-28T10:00:01Z\",\"caller\":\"bootstrap/infra.go:156\",\"msg\":\"database connection established\"}",
    truncated: false,
  },
  {
    id: "srv:lunafox-server:1735370402000000000:stdout:75ab9c10:000000",
    ts: "2024-12-28T10:00:02Z",
    tsNs: "1735370402000000000",
    stream: "stdout",
    line: "{\"level\":\"info\",\"timestamp\":\"2024-12-28T10:00:02Z\",\"caller\":\"bootstrap/infra.go:160\",\"msg\":\"redis connection established\"}",
    truncated: false,
  },
  {
    id: "srv:lunafox-server:1735370460000000000:stdout:e4b61a77:000000",
    ts: "2024-12-28T10:01:00Z",
    tsNs: "1735370460000000000",
    stream: "stdout",
    line: "{\"level\":\"info\",\"timestamp\":\"2024-12-28T10:01:00Z\",\"caller\":\"scan/handler.go:88\",\"msg\":\"scan created\",\"request.id\":\"req_mock_001\",\"scan.id\":101}",
    truncated: false,
  },
  {
    id: "srv:lunafox-server:1735370461000000000:stdout:layoutprobe:000000",
    ts: "2024-12-28T10:01:01Z",
    tsNs: "1735370461000000000",
    stream: "stdout",
    line: JSON.stringify({
      level: "info",
      timestamp: "2024-12-28T10:01:01Z",
      caller: "settings/system_logs_layout_probe.go:144",
      msg: "layout_overflow_probe captured long structured fields",
      "request.id": `mock_trace_${"a".repeat(96)}`,
      "runtime.configuration.digest": `sha256:${"f".repeat(128)}`,
      "target.scope": "subdomain-discovery-and-port-scan-with-very-long-unbroken-label.example.internal",
    }),
    truncated: false,
  },
  {
    id: "srv:lunafox-server:1735370462000000000:stderr:layoutprobe:000000",
    ts: "2024-12-28T10:01:02Z",
    tsNs: "1735370462000000000",
    stream: "stderr",
    line: `panic: layout_overflow_probe_${"x".repeat(180)} failed while rendering structured system log payload`,
    truncated: true,
  },
]

const generatedSystemLogItems = Array.from(
  { length: MOCK_SYSTEM_LOG_HISTORY_SIZE - mockSystemLogTailItems.length },
  (_, index): SystemLogItem => {
    const timestampMs = MOCK_SYSTEM_LOG_START_MS + index * 1000
    const timestamp = new Date(timestampMs).toISOString()
    const sequence = String(index).padStart(6, "0")
    const level = index % 41 === 0 ? "error" : index % 17 === 0 ? "warn" : index % 11 === 0 ? "debug" : "info"

    return {
      id: `srv:lunafox-server:${Math.floor(timestampMs / 1000)}000000000:stdout:mockperf:${sequence}`,
      ts: timestamp,
      tsNs: `${Math.floor(timestampMs / 1000)}000000000`,
      stream: level === "error" ? "stderr" : "stdout",
      line: JSON.stringify({
        level,
        timestamp,
        caller: `runtime/mock_worker_${index % 12}.go:${80 + index % 120}`,
        msg: `structured log performance fixture ${sequence}`,
        "request.id": `req_mock_${sequence}`,
        "trace.id": `trace_${String(index).padStart(24, "0")}`,
        "task.id": `tasks/${index % 300}`,
        "agent.id": `agents/${index % 24}`,
        component: index % 3 === 0 ? "scanner" : index % 3 === 1 ? "scheduler" : "api",
        "duration.ms": index % 2500,
        "http.method": index % 5 === 0 ? "POST" : "GET",
        "http.route": `/v1/mock/resources/${index % 120}`,
        "status.code": level === "error" ? 500 : 200,
      }),
      truncated: false,
    }
  },
)

export const mockSystemLogItems: SystemLogItem[] = [
  ...generatedSystemLogItems,
  ...mockSystemLogTailItems,
]

export function getMockSystemLogs(params?: {
  lines?: number
  cursor?: string
  direction?: "newer" | "older"
}): MockSystemLogEntriesResponse {
  const lines = params?.lines ?? 100
  if (!Number.isInteger(lines) || lines < 1) {
    throw new RangeError("Mock system log lines must be a positive integer.")
  }

  const followCursor = `mock-follow:${mockSystemLogItems.length}`
  if (params?.direction === "newer") {
    return {
      results: [],
      nextPageToken: followCursor,
      previousPageToken: "",
      hasOlder: false,
      hasNewer: false,
      caughtUp: true,
      gap: false,
      gapReason: "",
    }
  }

  const cursorEnd = params?.direction === "older"
    ? parseOlderCursor(params.cursor) ?? mockSystemLogItems.length
    : mockSystemLogItems.length
  const end = Math.min(Math.max(0, cursorEnd), mockSystemLogItems.length)
  const start = Math.max(0, end - lines)
  const logs = mockSystemLogItems.slice(start, end)

  return {
    results: logs,
    nextPageToken: followCursor,
    previousPageToken: start > 0 ? `mock-older:${start}` : "",
    hasOlder: start > 0,
    hasNewer: end < mockSystemLogItems.length,
    caughtUp: end === mockSystemLogItems.length,
    gap: false,
    gapReason: "",
  }
}

function parseOlderCursor(cursor: string | undefined): number | null {
  const match = cursor?.match(/^mock-older:(\d+)$/)
  return match ? Number.parseInt(match[1], 10) : null
}
