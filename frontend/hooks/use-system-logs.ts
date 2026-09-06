"use client"

import { useCallback, useEffect, useMemo, useRef } from "react"

import { SystemLogQueryError, systemLogService } from "@/services/system-log.service"
import { useToastMessages } from "@/lib/toast-helpers"
import {
  useLogStreamViewer,
  type FetchLogStreamParams,
  type LogStreamLineItem,
} from "@/hooks/use-log-stream-viewer"
import type { SystemLogItem } from "@/types/system-log.types"

const DEFAULT_LOG_WINDOW_SIZE = 100
const LOG_PAGE_SIZE = 500
const LOG_POLL_INTERVAL_MS = 2000

export type SystemLogLineItem = LogStreamLineItem

export function useSystemLogs(options?: {
  windowSize?: number
  enabled?: boolean
  autoRefresh?: boolean
}) {
  const hadErrorRef = useRef(false)
  const toastMessages = useToastMessages()
  const windowSize = options?.windowSize ?? DEFAULT_LOG_WINDOW_SIZE
  const enabled = options?.enabled ?? true

  const fetchLogs = useCallback(
    async (params: FetchLogStreamParams) => {
      const result = await systemLogService.fetchServerLogs({
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
    [],
  )

  const viewer = useLogStreamViewer({
    enabled,
    fetchLogs,
    windowSize,
    pollingEnabled: options?.autoRefresh ?? true,
    pollIntervalMs: LOG_POLL_INTERVAL_MS,
    pageSize: LOG_PAGE_SIZE,
  })

  useEffect(() => {
    if (viewer.phase === "error" && !hadErrorRef.current) {
      hadErrorRef.current = true
      toastMessages.error("toast.systemLog.fetch.error")
    }

    if (viewer.phase !== "error" && hadErrorRef.current) {
      hadErrorRef.current = false
      toastMessages.success("toast.systemLog.fetch.recovered")
    }
  }, [toastMessages, viewer.phase])

  const content = useMemo(() => {
    return viewer.lines.map((item) => item.line).join("\n")
  }, [viewer.lines])

  return {
    ...viewer,
    content,
    isLoading: viewer.phase === "initialLoading" || viewer.phase === "loadingOlder",
    isError: viewer.phase === "error",
    error: viewer.errorMessage ? new SystemLogQueryError(viewer.errorCode ?? "unknown", viewer.errorMessage) : null,
  }
}

function normalizeLines(items: SystemLogItem[]): SystemLogLineItem[] {
  return items.map((item) => ({
    id: item.id,
    ts: item.ts,
    stream: item.stream,
    line: item.line,
    truncated: item.truncated,
  }))
}
