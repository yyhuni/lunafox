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
import { AgentLogQueryError, agentService } from "@/services/agent.service"
import type { AgentLogItem } from "@/types/agent-log.types"
import type { Agent } from "@/types/agent.types"

const MAX_RENDER_LINES = 5000
const LOG_POLL_INTERVAL_MS = 2000

interface LogLineItem {
  id: string
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
  const [status, setStatus] = useState<"idle" | "connecting" | "streaming" | "done" | "error">("idle")
  const [errorMessage, setErrorMessage] = useState("")
  const [autoScroll, setAutoScroll] = useState(true)
  const [isPolling, setIsPolling] = useState(false)
  const [lineLimitReached, setLineLimitReached] = useState(false)

  const abortRef = useRef<AbortController | null>(null)
  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const isFetchingRef = useRef(false)
  const cursorRef = useRef("")
  const viewportRef = useRef<HTMLDivElement | null>(null)

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

  const stopPolling = useCallback(() => {
    abortRef.current?.abort()
    abortRef.current = null
    if (pollTimerRef.current !== null) {
      clearInterval(pollTimerRef.current)
      pollTimerRef.current = null
    }
    isFetchingRef.current = false
    setIsPolling(false)
  }, [])

  useEffect(() => {
    if (!open) {
      stopPolling()
      setStatus("idle")
      setErrorMessage("")
      cursorRef.current = ""
    }
  }, [open, stopPolling])

  useEffect(() => {
    return () => {
      stopPolling()
    }
  }, [stopPolling])

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

  const appendLogLines = useCallback((items: AgentLogItem[]) => {
    if (items.length === 0) {
      return
    }
    const mapped = items.map((item) => ({
      id: item.id,
      ts: item.ts,
      stream: item.stream,
      line: item.line,
      truncated: item.truncated,
    }))

    setLines((prev) => {
      if (mapped.length === 0) {
        return prev
      }
      const seen = new Set(prev.map((item) => item.id))
      const unique = mapped.filter((item) => {
        if (seen.has(item.id)) {
          return false
        }
        seen.add(item.id)
        return true
      })
      if (unique.length === 0) {
        return prev
      }
      const merged = [...prev, ...unique]
      if (merged.length <= MAX_RENDER_LINES) {
        return merged
      }
      setLineLimitReached(true)
      return merged.slice(merged.length - MAX_RENDER_LINES)
    })
  }, [])

  const mapErrorMessage = useCallback((code: string, fallback: string) => {
    const normalized = code.trim().toLowerCase()
    if (normalized === "container_not_found") return t("logs.errors.containerNotFound")
    if (normalized === "loki_unavailable") return t("logs.errors.lokiUnavailable")
    if (normalized === "query_timeout") return t("logs.errors.queryTimeout")
    if (normalized === "agent_not_found") return t("logs.errors.agentNotFound")
    if (normalized === "bad_request") return t("logs.errors.badRequest")
    if (normalized === "internal_error") return t("logs.errors.unknown")
    if (normalized === "agent_offline") return t("logs.errors.agentOffline")
    return fallback || t("logs.errors.unknown")
  }, [t])

  const pollOnce = useCallback(async (controller: AbortController) => {
    if (!agent || !container.trim()) {
      return
    }
    if (controller.signal.aborted || isFetchingRef.current) {
      return
    }

    isFetchingRef.current = true
    try {
      const result = await agentService.fetchAgentLogs({
        agentId: agent.id,
        container: container.trim(),
        limit: 200,
        cursor: cursorRef.current || undefined,
        signal: controller.signal,
      })
      if (controller.signal.aborted) {
        return
      }

      if (result.logs.length > 0) {
        appendLogLines(result.logs)
      }

      const nextCursor = result.nextCursor.trim()
      if (nextCursor) {
        cursorRef.current = nextCursor
      }

      setStatus("streaming")
      setErrorMessage("")
    } catch (error) {
      if (isAbortError(error)) {
        return
      }

      setStatus("error")
      if (error instanceof AgentLogQueryError) {
        const reason = mapErrorMessage(error.code, error.message)
        setErrorMessage(`[${error.code}] ${reason}`)
      } else if (error instanceof Error) {
        setErrorMessage(error.message)
      } else {
        setErrorMessage(t("logs.openFailed"))
      }
    } finally {
      isFetchingRef.current = false
    }
  }, [agent, appendLogLines, mapErrorMessage, t])

  const startPolling = useCallback(async () => {
    if (!agent || !container.trim()) {
      return
    }

    stopPolling()
    cursorRef.current = ""
    setLines([])
    setStatus("connecting")
    setErrorMessage("")
    setIsPolling(true)
    setLineLimitReached(false)

    const controller = new AbortController()
    abortRef.current = controller

    await pollOnce(controller)
    if (controller.signal.aborted) {
      return
    }

    setStatus((prev) => (prev === "error" ? prev : "streaming"))
    pollTimerRef.current = setInterval(() => {
      void pollOnce(controller)
    }, LOG_POLL_INTERVAL_MS)
  }, [agent, pollOnce, stopPolling])

  useEffect(() => {
    if (!open || !agent) {
      return
    }
    void startPolling()
  }, [open, agent, startPolling])

  const onToggleOpen = useCallback((nextOpen: boolean) => {
    if (!nextOpen) {
      stopPolling()
      setStatus("done")
    }
    onOpenChange(nextOpen)
  }, [onOpenChange, stopPolling])

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
              {errorMessage || (isPolling ? t("logs.status.streaming") : t("logs.ready"))}
            </span>
          </div>
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
  if (error instanceof Error) {
    return error.name === "AbortError"
  }
  return false
}
