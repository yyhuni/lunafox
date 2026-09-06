import React from "react"

import type { ScanLog } from "@/types/scan.types"
import { formatLocalTimestampSeconds } from "@/lib/log-time"

interface ScanLogListStateOptions {
  logs: ScanLog[]
}

export function formatScanLogLine(log: ScanLog) {
  const time = formatLocalTimestampSeconds(log.createdAt)
  const levelTag = log.level.toUpperCase()
  return `[${time}] [${levelTag}] ${log.content}`
}

export function buildScanLogContent(logs: ScanLog[]) {
  return logs.map(formatScanLogLine).join("\n")
}

export function useScanLogListState({ logs }: ScanLogListStateOptions) {
  const contentRef = React.useRef("")
  const lastLogCountRef = React.useRef(0)
  const lastLogIdRef = React.useRef<number | null>(null)
  const firstLogIdRef = React.useRef<number | null>(null)

  const content = React.useMemo(() => {
    if (logs.length === 0) {
      contentRef.current = ""
      lastLogCountRef.current = 0
      lastLogIdRef.current = null
      firstLogIdRef.current = null
      return ""
    }

    const lastLog = logs[logs.length - 1]
    const firstLog = logs[0]

    const shouldRebuild =
      lastLogIdRef.current === null ||
      logs.length < lastLogCountRef.current ||
      (firstLogIdRef.current !== null && firstLog?.id !== firstLogIdRef.current)

    if (!shouldRebuild) {
      if (logs.length === lastLogCountRef.current && lastLog?.id === lastLogIdRef.current) {
        return contentRef.current
      }

      const lastIndex = logs.findIndex((log) => log.id === lastLogIdRef.current)
      if (lastIndex !== -1) {
        const newLogs = logs.slice(lastIndex + 1)
        if (newLogs.length > 0) {
          const appended = newLogs.map(formatScanLogLine).join("\n")
          contentRef.current = contentRef.current ? `${contentRef.current}\n${appended}` : appended
        }
        lastLogCountRef.current = logs.length
        lastLogIdRef.current = lastLog?.id ?? null
        firstLogIdRef.current = firstLog?.id ?? null
        return contentRef.current
      }
    }

    const newContent = buildScanLogContent(logs)

    contentRef.current = newContent
    lastLogCountRef.current = logs.length
    lastLogIdRef.current = lastLog?.id ?? null
    firstLogIdRef.current = firstLog?.id ?? null
    return newContent
  }, [logs])

  return {
    content,
  }
}

export type ScanLogListState = ReturnType<typeof useScanLogListState>
