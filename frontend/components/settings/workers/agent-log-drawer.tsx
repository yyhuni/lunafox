"use client"

import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslations } from "next-intl"
import { IconTerminal } from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { cn } from "@/lib/utils"
import { AgentLogStreamError, agentService } from "@/services/agent.service"
import type { Agent } from "@/types/agent.types"
import type { AgentLogStreamEvent } from "@/types/agent-log.types"

const MAX_RENDER_LINES = 5000
const LOG_FLUSH_INTERVAL_MS = 80

interface LogLineItem {
  id: number
  ts: string
  stream: string
  line: string
  truncated: boolean
}

interface AgentLogDrawerProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  agent: Agent | null
}

export function AgentLogDrawer({ open, onOpenChange, agent }: AgentLogDrawerProps) {
  const t = useTranslations("settings.workers")

  const container = "lunafox-agent"
  const [lines, setLines] = useState<LogLineItem[]>([])
  const [requestId, setRequestId] = useState("")
  const [status, setStatus] = useState<"idle" | "connecting" | "streaming" | "done" | "error">("idle")
  const [statusReason, setStatusReason] = useState("")
  const [errorMessage, setErrorMessage] = useState("")
  const [autoScroll, setAutoScroll] = useState(true)
  const [isStreaming, setIsStreaming] = useState(false)
  const [lineLimitReached, setLineLimitReached] = useState(false)

  const abortRef = useRef<AbortController | null>(null)
  const viewportRef = useRef<HTMLDivElement | null>(null)
  const lineIdRef = useRef(0)
  const lineCountRef = useRef(0)
  const bufferedLinesRef = useRef<LogLineItem[]>([])
  const flushTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const statusText = useMemo(() => {
    if (status === "connecting") return t("logs.status.connecting")
    if (status === "streaming") return t("logs.status.streaming")
    if (status === "done") return t("logs.status.done")
    if (status === "error") return t("logs.status.error")
    return t("logs.status.idle")
  }, [status, t])

  const statusDotClassName = useMemo(() => {
    if (status === "streaming") return "bg-emerald-500"
    if (status === "connecting") return "bg-sky-500 animate-pulse"
    if (status === "done") return "bg-zinc-500"
    if (status === "error") return "bg-rose-500"
    return "bg-zinc-400"
  }, [status])

  const statusTextClassName = useMemo(() => {
    if (status === "streaming") return "text-emerald-600"
    if (status === "connecting") return "text-sky-600"
    if (status === "done") return "text-zinc-600"
    if (status === "error") return "text-rose-600"
    return "text-muted-foreground"
  }, [status])

  const resetBufferedLines = useCallback(() => {
    bufferedLinesRef.current = []
    if (flushTimerRef.current !== null) {
      clearTimeout(flushTimerRef.current)
      flushTimerRef.current = null
    }
  }, [])

  const flushBufferedLines = useCallback(() => {
    if (flushTimerRef.current !== null) {
      clearTimeout(flushTimerRef.current)
      flushTimerRef.current = null
    }

    if (bufferedLinesRef.current.length === 0) {
      return
    }

    const chunk = bufferedLinesRef.current
    bufferedLinesRef.current = []

    if (lineCountRef.current + chunk.length > MAX_RENDER_LINES) {
      setLineLimitReached(true)
    }

    setLines((prev) => {
      const merged = [...prev, ...chunk]
      if (merged.length <= MAX_RENDER_LINES) {
        lineCountRef.current = merged.length
        return merged
      }
      const sliced = merged.slice(merged.length - MAX_RENDER_LINES)
      lineCountRef.current = sliced.length
      return sliced
    })
  }, [])

  const scheduleFlushBufferedLines = useCallback(() => {
    if (flushTimerRef.current !== null) {
      return
    }
    flushTimerRef.current = setTimeout(() => {
      flushBufferedLines()
    }, LOG_FLUSH_INTERVAL_MS)
  }, [flushBufferedLines])

  const stopStream = useCallback(() => {
    abortRef.current?.abort()
    abortRef.current = null
    resetBufferedLines()
    setIsStreaming(false)
  }, [resetBufferedLines])

  useEffect(() => {
    if (!open) {
      stopStream()
      setStatus("idle")
      setStatusReason("")
      setRequestId("")
    }
  }, [open, stopStream])

  useEffect(() => {
    return () => {
      stopStream()
    }
  }, [stopStream])

  useEffect(() => {
    if (!autoScroll) {
      return
    }
    const viewport = viewportRef.current
    if (!viewport) {
      return
    }
    viewport.scrollTop = viewport.scrollHeight
  }, [lines, autoScroll])

  const appendLogLine = useCallback((event: Extract<AgentLogStreamEvent, { type: "log" }>) => {
    const nextItem: LogLineItem = {
      id: ++lineIdRef.current,
      ts: event.ts,
      stream: event.stream,
      line: event.line,
      truncated: event.truncated,
    }
    bufferedLinesRef.current.push(nextItem)
    scheduleFlushBufferedLines()
  }, [scheduleFlushBufferedLines])

  const mapErrorMessage = useCallback((code: string, fallback: string) => {
    const normalized = code.trim().toLowerCase()
    if (normalized === "container_not_found") return t("logs.errors.containerNotFound")
    if (normalized === "docker_unavailable") return t("logs.errors.dockerUnavailable")
    if (normalized === "docker_api_error") return t("logs.errors.dockerApiError")
    if (normalized === "agent_timeout") return t("logs.errors.agentTimeout")
    if (normalized === "ws_send_failed") return t("logs.errors.wsSendFailed")
    if (normalized === "agent_offline") return t("logs.errors.agentOffline")
    if (normalized === "bad_request") return t("logs.errors.badRequest")
    if (normalized === "stream_unavailable") return t("logs.errors.streamUnavailable")
    return fallback || t("logs.errors.unknown")
  }, [t])

  const handleStreamEvent = useCallback((event: AgentLogStreamEvent) => {
    if (event.type === "ping") {
      return
    }

    setRequestId(event.requestId)

    if (event.type === "status") {
      if (event.status === "started") {
        setStatus("streaming")
      }
      return
    }

    if (event.type === "log") {
      appendLogLine(event)
      return
    }

    if (event.type === "error") {
      flushBufferedLines()
      setStatus("error")
      const reason = mapErrorMessage(event.code, event.message)
      setErrorMessage(`[${event.code}] ${reason}`)
      return
    }

    if (event.type === "done") {
      flushBufferedLines()
      setStatus("done")
      setStatusReason(event.reason)
      setIsStreaming(false)
    }
  }, [appendLogLine, flushBufferedLines, mapErrorMessage])

  const startStream = useCallback(async () => {
    if (!agent || !container.trim()) {
      return
    }

    stopStream()
    lineIdRef.current = 0
    lineCountRef.current = 0
    resetBufferedLines()
    setLines([])
    setRequestId("")
    setStatus("connecting")
    setStatusReason("")
    setErrorMessage("")
    setIsStreaming(true)
    setLineLimitReached(false)

    const controller = new AbortController()
    abortRef.current = controller

    try {
      await agentService.streamAgentLogs({
        agentId: agent.id,
        container: container.trim(),
        tail: 200,
        follow: true,
        timestamps: true,
        signal: controller.signal,
        onEvent: handleStreamEvent,
      })
      if (!controller.signal.aborted) {
        setStatus((prev) => (prev === "error" ? prev : "done"))
      }
    } catch (error) {
      if (isAbortError(error)) {
        return
      }

      flushBufferedLines()
      setStatus("error")
      if (error instanceof AgentLogStreamError) {
        const reason = mapErrorMessage(error.code, error.message)
        setErrorMessage(`[${error.code}] ${reason}`)
      } else if (error instanceof Error) {
        setErrorMessage(error.message)
      } else {
        setErrorMessage(t("logs.openFailed"))
      }
    } finally {
      if (abortRef.current === controller) {
        abortRef.current = null
      }
      setIsStreaming(false)
    }
  }, [agent, flushBufferedLines, handleStreamEvent, mapErrorMessage, resetBufferedLines, stopStream, t])

  useEffect(() => {
    if (!open || !agent) {
      return
    }
    void startStream()
  }, [open, agent, startStream])

  const onToggleOpen = useCallback((nextOpen: boolean) => {
    if (!nextOpen) {
      stopStream()
    }
    onOpenChange(nextOpen)
  }, [onOpenChange, stopStream])

  const handleViewportScroll = useCallback(() => {
    const viewport = viewportRef.current
    if (!viewport) {
      return
    }
    const distanceToBottom = viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight
    if (distanceToBottom > 24 && autoScroll) {
      setAutoScroll(false)
      return
    }
    if (distanceToBottom <= 8 && !autoScroll) {
      setAutoScroll(true)
    }
  }, [autoScroll])

  return (
    <Sheet open={open} onOpenChange={onToggleOpen}>
      <SheetContent className="w-full sm:max-w-4xl p-0 gap-0">
        <SheetHeader className="border-b px-4 py-2">
          <div className="flex items-center gap-3">
            <div className="min-w-0">
              <SheetTitle className="flex items-center gap-2 text-base">
                <IconTerminal className="h-4 w-4" />
                {t("logs.title")}
              </SheetTitle>
              <p className="mt-0.5 truncate text-xs text-muted-foreground">
                {agent
                  ? `${agent.name} (${agent.ipAddress || t("unknownIp")}) · ${container}`
                  : t("logs.noAgent")}
              </p>
            </div>
          </div>
        </SheetHeader>

        <div className="flex-1 min-h-0 bg-[#0b1320] text-[#e5e7eb]">
          <div
            ref={viewportRef}
            onScroll={handleViewportScroll}
            className="h-full overflow-auto px-4 py-3 font-mono text-xs leading-5"
          >
            {lines.length === 0 ? (
              <div className="text-[#6b7280]">{t("logs.empty")}</div>
            ) : (
              <>
                {lineLimitReached && (
                  <div className="mb-2 text-[11px] text-amber-300/90">
                    {t("logs.lineLimitHint", { max: MAX_RENDER_LINES })}
                  </div>
                )}
                {lines.map((line) => (
                  <div key={line.id} className="whitespace-pre-wrap break-words">
                    <span className="text-[#6b7280] mr-2">{formatLogTime(line.ts)}</span>
                    <span
                      className={cn(
                        "mr-2 uppercase font-semibold",
                        isErrorStream(line.stream) ? "text-[#f87171]" : "text-[#34d399]"
                      )}
                    >
                      {isErrorStream(line.stream) ? "stderr" : "stdout"}
                    </span>
                    <span className={isErrorStream(line.stream) ? "text-[#fecaca]" : "text-[#d1fae5]"}>
                      {line.line || " "}
                    </span>
                    {line.truncated && (
                      <Badge variant="outline" className="ml-2 h-4 px-1 text-[10px] text-amber-400 border-amber-400/40">
                        {t("logs.truncated")}
                      </Badge>
                    )}
                  </div>
                ))}
              </>
            )}
          </div>
        </div>

        <div className="border-t px-4 py-2 text-xs bg-muted/30 flex items-center justify-between gap-3">
          <div className="min-w-0 flex items-center gap-2">
            <span
              className={cn(
                "h-2 w-2 shrink-0 rounded-full",
                statusDotClassName
              )}
              aria-hidden="true"
            />
            <span className={cn("shrink-0 text-[11px] font-medium", statusTextClassName)}>
              {statusText}
            </span>
            <span className="truncate text-muted-foreground">
              {errorMessage || statusReason || t("logs.ready")}
            </span>
          </div>
          {requestId && (
            <span className="font-mono text-muted-foreground/80">requestId: {requestId}</span>
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}

function formatLogTime(raw: string): string {
  const parsed = new Date(raw)
  if (Number.isNaN(parsed.getTime())) {
    return "--:--:--"
  }
  return parsed.toLocaleTimeString([], { hour12: false })
}

function isErrorStream(stream: string): boolean {
  return stream.toLowerCase() === "stderr"
}

function isAbortError(error: unknown): boolean {
  if (error instanceof DOMException) {
    return error.name === "AbortError"
  }
  return false
}
