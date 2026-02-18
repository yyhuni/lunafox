import React from "react"

import type { ScanLog } from "@/services/scan.service"

interface ScanLogListStateOptions {
  logs: ScanLog[]
}

function formatTime(isoString: string): string {
  try {
    const date = new Date(isoString)
    const h = String(date.getHours()).padStart(2, "0")
    const m = String(date.getMinutes()).padStart(2, "0")
    const s = String(date.getSeconds()).padStart(2, "0")
    return `${h}:${m}:${s}`
  } catch {
    return isoString
  }
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
          const appended = newLogs
            .map((log) => {
              const time = formatTime(log.createdAt)
              const levelTag = log.level.toUpperCase()
              return `[${time}] [${levelTag}] ${log.content}`
            })
            .join("\n")
          contentRef.current = contentRef.current ? `${contentRef.current}\n${appended}` : appended
        }
        lastLogCountRef.current = logs.length
        lastLogIdRef.current = lastLog?.id ?? null
        firstLogIdRef.current = firstLog?.id ?? null
        return contentRef.current
      }
    }

    const newContent = logs
      .map((log) => {
        const time = formatTime(log.createdAt)
        const levelTag = log.level.toUpperCase()
        return `[${time}] [${levelTag}] ${log.content}`
      })
      .join("\n")

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
