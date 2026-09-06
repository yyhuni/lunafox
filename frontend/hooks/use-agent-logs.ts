"use client"

import { useCallback } from "react"
import { agentService } from "@/services/agent.service"
import type { AgentLogItem } from "@/types/agent-log.types"
import type { Agent } from "@/types/agent.types"
import {
  useLogStreamViewer,
  type FetchLogStreamParams,
  type LogStreamLineItem,
  type LogStreamViewerPhase,
} from "@/hooks/use-log-stream-viewer"

const LOG_POLL_INTERVAL_MS = 2000
const LOG_PAGE_SIZE = 500

export type AgentLogLineItem = LogStreamLineItem
export type AgentLogViewerPhase = LogStreamViewerPhase

export function useAgentLogs({
  open,
  agentNode,
  container,
  windowSize = 100,
  pollIntervalMs = LOG_POLL_INTERVAL_MS,
  pollingEnabled = true,
}: {
  open: boolean
  agentNode: Agent | null
  container: string
  windowSize?: number
  pollIntervalMs?: number
  pollingEnabled?: boolean
}) {
  const fetchLogs = useCallback(
    async (params: FetchLogStreamParams) => {
      if (!agentNode || !container.trim()) {
        return emptyAgentLogResponse()
      }

      const result = await agentService.fetchDistributedLogs({
        agentNodeId: agentNode.id,
        container: container.trim(),
        limit: params.limit,
        cursor: params.cursor,
        direction: params.direction,
        signal: params.signal,
      })

      return {
        ...result,
        logs: normalizeLines(result.logs),
      }
    },
    [agentNode, container],
  )

  return useLogStreamViewer({
    enabled: open && Boolean(agentNode) && Boolean(container.trim()),
    fetchLogs,
    windowSize,
    pollIntervalMs,
    pollingEnabled,
    pageSize: LOG_PAGE_SIZE,
  })
}

function normalizeLines(items: AgentLogItem[]): AgentLogLineItem[] {
  return items.map((item) => ({
    id: item.id,
    ts: item.ts,
    stream: item.stream,
    line: item.line,
    truncated: item.truncated,
  }))
}

function emptyAgentLogResponse() {
  return {
    logs: [] as AgentLogLineItem[],
    nextCursor: "",
    previousCursor: "",
    hasOlder: false,
    hasNewer: false,
    caughtUp: false,
    gap: false,
    gapReason: "",
  }
}
