"use client"

import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react"

const DEFAULT_LOG_WINDOW_SIZE = 100
const LOG_PAGE_SIZE = 500
const LOG_POLL_INTERVAL_MS = 2000
const MAX_HISTORY_RETRIES = 3
const HISTORY_RETRY_BASE_MS = 250
const VIEWPORT_BOTTOM_THRESHOLD_PX = 24
const AUTO_SCROLL_SETTLE_MS = 100
const AUTO_SCROLL_RELEASE_MS = 50

export interface LogStreamLineItem {
  id: string
  ts: string
  stream: string
  line: string
  truncated: boolean
}

export interface LogStreamResponse<TLog extends LogStreamLineItem = LogStreamLineItem> {
  logs: TLog[]
  nextCursor: string
  previousCursor: string
  hasOlder: boolean
  hasNewer: boolean
  caughtUp: boolean
  gap: boolean
  gapReason: string
}

export interface FetchLogStreamParams {
  limit: number
  cursor?: string
  direction?: "newer" | "older"
  signal?: AbortSignal
}

export type LogStreamViewerPhase =
  | "idle"
  | "initialLoading"
  | "ready"
  | "loadingOlder"
  | "following"
  | "catchingUp"
  | "paused"
  | "error"

interface MergeResult<TLine extends LogStreamLineItem> {
  lines: TLine[]
  trimmedBefore: boolean
}

export function useLogStreamViewer<TLine extends LogStreamLineItem = LogStreamLineItem>({
  enabled,
  fetchLogs,
  windowSize = DEFAULT_LOG_WINDOW_SIZE,
  pollIntervalMs = LOG_POLL_INTERVAL_MS,
  pollingEnabled = true,
  pageSize = LOG_PAGE_SIZE,
}: {
  enabled: boolean
  fetchLogs: (params: FetchLogStreamParams) => Promise<LogStreamResponse<TLine>>
  windowSize?: number
  pollIntervalMs?: number
  pollingEnabled?: boolean
  pageSize?: number
}) {
  const normalizedWindowSize = Math.max(1, windowSize)
  const normalizedPageSize = Math.min(Math.max(1, pageSize), LOG_PAGE_SIZE)
  const [lines, setLines] = useState<TLine[]>([])
  const [phase, setPhase] = useState<LogStreamViewerPhase>("idle")
  const [errorCode, setErrorCode] = useState<string | null>(null)
  const [errorMessage, setErrorMessage] = useState("")
  const [autoScroll, setAutoScroll] = useState(true)
  const [trimmedBefore, setTrimmedBefore] = useState(false)
  const [trimmedAfter, setTrimmedAfter] = useState(false)
  const [hasOlder, setHasOlder] = useState(false)
  const [hasNewer, setHasNewer] = useState(false)
  const [caughtUp, setCaughtUp] = useState(false)
  const [gap, setGap] = useState(false)
  const [gapReason, setGapReason] = useState("")

  const viewportRef = useRef<HTMLDivElement | null>(null)
  const linesRef = useRef<TLine[]>([])
  const autoScrollRef = useRef(true)
  const autoRefreshRef = useRef(pollingEnabled)
  const previousAutoRefreshRef = useRef(pollingEnabled)
  const generationRef = useRef(0)
  const sourceRef = useRef(fetchLogs)
  const previousWindowSizeRef = useRef(normalizedWindowSize)
  const windowSizeRef = useRef(normalizedWindowSize)
  const initializedRef = useRef(false)
  // This spans the first latest page and optional older hydration. A boolean
  // alone is unsafe because a canceled generation can settle after its successor.
  const initializationGenerationRef = useRef<number | null>(null)
  const historyHydrationGenerationRef = useRef<number | null>(null)
  const historyAbortRef = useRef<AbortController | null>(null)
  const followAbortRef = useRef<AbortController | null>(null)
  const retryTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const pollTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const autoScrollSettleTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const programmaticAutoScrollRef = useRef(false)
  const viewportSizeRef = useRef<{ width: number; height: number } | null>(null)
  const followCursorRef = useRef("")
  const olderCursorRef = useRef("")
  const gapRef = useRef(false)
  // A failed history hydration leaves the viewer with a valid partial window.
  // Keep that warning visible while follow continues so a successful live page
  // cannot make the missing history look complete.
  const errorOriginRef = useRef<"history" | null>(null)
  const runFollowRef = useRef<(generation: number) => Promise<void>>(async () => {})

  windowSizeRef.current = normalizedWindowSize

  const clearRetryTimer = useCallback(() => {
    if (retryTimerRef.current !== null) {
      clearTimeout(retryTimerRef.current)
      retryTimerRef.current = null
    }
  }, [])

  const clearPollTimer = useCallback(() => {
    if (pollTimerRef.current !== null) {
      clearTimeout(pollTimerRef.current)
      pollTimerRef.current = null
    }
  }, [])

  const clearAutoScrollSettle = useCallback(() => {
    if (autoScrollSettleTimerRef.current !== null) {
      clearTimeout(autoScrollSettleTimerRef.current)
      autoScrollSettleTimerRef.current = null
    }
    programmaticAutoScrollRef.current = false
  }, [])

  const scheduleAutoScrollSettlement = useCallback(() => {
    const viewport = viewportRef.current
    if (!viewport || !autoScrollRef.current) return

    clearAutoScrollSettle()
    programmaticAutoScrollRef.current = true
    viewport.scrollTop = viewport.scrollHeight
    autoScrollSettleTimerRef.current = setTimeout(() => {
      if (autoScrollRef.current && viewportRef.current) {
        viewportRef.current.scrollTop = viewportRef.current.scrollHeight
      }
      autoScrollSettleTimerRef.current = setTimeout(() => {
        programmaticAutoScrollRef.current = false
        autoScrollSettleTimerRef.current = null
      }, AUTO_SCROLL_RELEASE_MS)
    }, AUTO_SCROLL_SETTLE_MS)
  }, [clearAutoScrollSettle])

  const setAutoScrollState = useCallback((value: boolean) => {
    if (!value) clearAutoScrollSettle()
    autoScrollRef.current = value
    setAutoScroll(value)
  }, [clearAutoScrollSettle])

  const commitLines = useCallback((next: TLine[]) => {
    linesRef.current = next
    setLines(next)
  }, [])

  const resetViewerState = useCallback(() => {
    commitLines([])
    setPhase("idle")
    errorOriginRef.current = null
    setErrorCode(null)
    setErrorMessage("")
    setAutoScrollState(true)
    setTrimmedBefore(false)
    setTrimmedAfter(false)
    setHasOlder(false)
    setHasNewer(false)
    setCaughtUp(false)
    setGap(false)
    setGapReason("")
    gapRef.current = false
    followCursorRef.current = ""
    olderCursorRef.current = ""
  }, [commitLines, setAutoScrollState])

  const cancelFollow = useCallback(() => {
    clearPollTimer()
    followAbortRef.current?.abort()
    followAbortRef.current = null
  }, [clearPollTimer])

  const cancelHistoryHydration = useCallback(() => {
    clearRetryTimer()
    historyAbortRef.current?.abort()
    historyAbortRef.current = null
    historyHydrationGenerationRef.current = null
  }, [clearRetryTimer])

  const cancelGeneration = useCallback(() => {
    generationRef.current += 1
    initializationGenerationRef.current = null
    clearAutoScrollSettle()
    cancelHistoryHydration()
    cancelFollow()
  }, [cancelFollow, cancelHistoryHydration, clearAutoScrollSettle])

  const mergeUnique = useCallback((previous: TLine[], incoming: TLine[], prepend: boolean): MergeResult<TLine> => {
    const seen = new Set(previous.map((item) => item.id))
    const unique = incoming.filter((item) => {
      if (seen.has(item.id)) return false
      seen.add(item.id)
      return true
    })
    if (unique.length === 0 && previous.length <= windowSizeRef.current) {
      return { lines: previous, trimmedBefore: false }
    }
    const merged = prepend ? [...unique, ...previous] : [...previous, ...unique]
    if (merged.length <= windowSizeRef.current) {
      return { lines: merged, trimmedBefore: false }
    }
    return { lines: merged.slice(-windowSizeRef.current), trimmedBefore: true }
  }, [])

  const applyLatestMetadata = useCallback((result: LogStreamResponse<TLine>) => {
    followCursorRef.current = result.nextCursor.trim()
    olderCursorRef.current = result.previousCursor.trim()
    setHasOlder(result.hasOlder)
    setHasNewer(result.hasNewer)
    setCaughtUp(result.caughtUp)
    setGap(result.gap)
    setGapReason(result.gapReason)
    gapRef.current = result.gap
  }, [])

  const applyOlderMetadata = useCallback((result: LogStreamResponse<TLine>) => {
    olderCursorRef.current = result.previousCursor.trim()
    setHasOlder(result.hasOlder)
    setGap(result.gap)
    setGapReason(result.gapReason)
    gapRef.current = result.gap
  }, [])

  const applyFollowMetadata = useCallback((result: LogStreamResponse<TLine>) => {
    followCursorRef.current = result.nextCursor.trim()
    setHasNewer(result.hasNewer)
    setCaughtUp(result.caughtUp)
    setGap(result.gap)
    setGapReason(result.gapReason)
    gapRef.current = result.gap
  }, [])

  const setError = useCallback((error: unknown, origin: "history" | "live" = "live") => {
    if (origin === "live" && errorOriginRef.current === "history") {
      setPhase("error")
      return
    }
    errorOriginRef.current = origin === "history" ? "history" : null
    setPhase("error")
    if (hasLogQueryErrorShape(error)) {
      setErrorCode(error.code)
      setErrorMessage(error.message)
      return
    }
    setErrorCode(null)
    setErrorMessage(error instanceof Error ? error.message : "")
  }, [])

  const waitForRetry = useCallback((signal: AbortSignal, delayMs: number) => {
    return new Promise<boolean>((resolve) => {
      const finish = (value: boolean) => {
        signal.removeEventListener("abort", onAbort)
        retryTimerRef.current = null
        resolve(value)
      }
      const onAbort = () => {
        if (retryTimerRef.current !== null) clearTimeout(retryTimerRef.current)
        finish(false)
      }
      retryTimerRef.current = setTimeout(() => finish(true), delayMs)
      signal.addEventListener("abort", onAbort, { once: true })
    })
  }, [])

  const schedulePolling = useCallback((generation: number) => {
    clearPollTimer()
    if (!enabled || !autoRefreshRef.current || generationRef.current !== generation) return
    pollTimerRef.current = setTimeout(() => {
      pollTimerRef.current = null
      void runFollowRef.current(generation)
    }, pollIntervalMs)
  }, [clearPollTimer, enabled, pollIntervalMs])

  const runFollow = useCallback(async (generation: number) => {
    if (
      !enabled ||
      !autoRefreshRef.current ||
      generationRef.current !== generation ||
      initializationGenerationRef.current === generation ||
      followAbortRef.current
    ) {
      return
    }
    if (!followCursorRef.current.trim() && gapRef.current) {
      setPhase("ready")
      return
    }
    const controller = new AbortController()
    followAbortRef.current = controller
    setPhase("catchingUp")
    try {
      while (
        !controller.signal.aborted &&
        autoRefreshRef.current &&
        generationRef.current === generation
      ) {
        const cursor = followCursorRef.current.trim()
        const result = await fetchLogs(cursor
          ? {
              limit: normalizedPageSize,
              cursor,
              direction: "newer",
              signal: controller.signal,
            }
          : {
              limit: normalizedPageSize,
              signal: controller.signal,
            })
        if (controller.signal.aborted || generationRef.current !== generation) return

        const merged = mergeUnique(linesRef.current, result.logs, false)
        if (merged.lines !== linesRef.current) commitLines(merged.lines)
        if (merged.trimmedBefore) setTrimmedBefore(true)
        if (cursor) {
          applyFollowMetadata(result)
        } else {
          // An empty latest response has no opaque follow token. Re-checking
          // latest on the next interval is the only safe way to establish one.
          applyLatestMetadata(result)
        }
        if (errorOriginRef.current !== "history") {
          setErrorCode(null)
          setErrorMessage("")
        }

        if (!result.hasNewer || !followCursorRef.current.trim()) break
      }
      if (!controller.signal.aborted && autoRefreshRef.current && generationRef.current === generation) {
        setPhase("following")
        schedulePolling(generation)
      }
    } catch (error) {
      if (!isAbortError(error) && !controller.signal.aborted && generationRef.current === generation) {
        setError(error)
        if (isRecoverableLogQueryError(error) && autoRefreshRef.current) {
          schedulePolling(generation)
        }
      }
    } finally {
      if (followAbortRef.current === controller) followAbortRef.current = null
    }
  }, [applyFollowMetadata, applyLatestMetadata, commitLines, enabled, fetchLogs, mergeUnique, normalizedPageSize, schedulePolling, setError])

  runFollowRef.current = runFollow

  const hydrateHistory = useCallback(async (generation: number, controller: AbortController) => {
    historyHydrationGenerationRef.current = generation
    try {
      while (
        !controller.signal.aborted &&
        generationRef.current === generation &&
        linesRef.current.length < windowSizeRef.current &&
        olderCursorRef.current.trim()
      ) {
        const cursor = olderCursorRef.current.trim()
        let attempt = 0
        let result: LogStreamResponse<TLine> | null = null
        while (!controller.signal.aborted && generationRef.current === generation) {
          try {
            result = await fetchLogs({
              limit: normalizedPageSize,
              cursor,
              direction: "older",
              signal: controller.signal,
            })
            break
          } catch (error) {
            if (isAbortError(error) || controller.signal.aborted || generationRef.current !== generation) return
            if (!isRecoverableLogQueryError(error) || attempt >= MAX_HISTORY_RETRIES) {
              setError(error, "history")
              return
            }
            attempt += 1
            const shouldRetry = await waitForRetry(controller.signal, HISTORY_RETRY_BASE_MS * 2 ** (attempt - 1))
            if (!shouldRetry) return
          }
        }
        if (!result || controller.signal.aborted || generationRef.current !== generation) return

        const merged = mergeUnique(linesRef.current, result.logs, true)
        if (merged.lines !== linesRef.current) commitLines(merged.lines)
        if (merged.trimmedBefore) setTrimmedBefore(true)
        applyOlderMetadata(result)
        if (errorOriginRef.current !== "history") {
          setErrorCode(null)
          setErrorMessage("")
        }
        if (result.gap || !result.hasOlder || !result.previousCursor.trim()) break
      }
    } finally {
      if (historyAbortRef.current === controller) historyAbortRef.current = null
      if (historyHydrationGenerationRef.current === generation) {
        historyHydrationGenerationRef.current = null
      }
    }
  }, [applyOlderMetadata, commitLines, fetchLogs, mergeUnique, normalizedPageSize, setError, waitForRetry])

  const startGeneration = useCallback(async (preserveVisibleLines: boolean) => {
    cancelGeneration()
    const generation = generationRef.current + 1
    generationRef.current = generation
    initializationGenerationRef.current = generation
    if (!preserveVisibleLines) {
      resetViewerState()
    } else {
      setPhase("initialLoading")
      errorOriginRef.current = null
      setErrorCode(null)
      setErrorMessage("")
      setTrimmedAfter(false)
    }

    const controller = new AbortController()
    historyAbortRef.current = controller
    setPhase("initialLoading")
    try {
      const result = await fetchLogs({ limit: normalizedPageSize, signal: controller.signal })
      if (controller.signal.aborted || generationRef.current !== generation) return
      const latest = result.logs.slice(-windowSizeRef.current)
      commitLines(latest)
      setTrimmedBefore(result.logs.length > latest.length)
      setTrimmedAfter(false)
      applyLatestMetadata(result)
      errorOriginRef.current = null
      setErrorCode(null)
      setErrorMessage("")

      let hydratedHistory = false
      if (latest.length < windowSizeRef.current && result.hasOlder && result.previousCursor.trim() && !result.gap) {
        hydratedHistory = true
        setPhase("loadingOlder")
        await hydrateHistory(generation, controller)
      }
      if (controller.signal.aborted || generationRef.current !== generation) return
      if (initializationGenerationRef.current === generation) {
        initializationGenerationRef.current = null
      }
      if (autoRefreshRef.current) {
        if (!followCursorRef.current.trim() && gapRef.current) {
          setPhase("ready")
        } else if (hydratedHistory || result.hasNewer) {
          await runFollowRef.current(generation)
        } else {
          setPhase("following")
          schedulePolling(generation)
        }
      } else {
        setPhase(errorOriginRef.current === "history" ? "error" : "ready")
      }
    } catch (error) {
      if (!isAbortError(error) && !controller.signal.aborted && generationRef.current === generation) {
        setError(error)
      }
    } finally {
      if (historyAbortRef.current === controller) historyAbortRef.current = null
      if (initializationGenerationRef.current === generation) {
        initializationGenerationRef.current = null
      }
    }
  }, [applyLatestMetadata, cancelGeneration, commitLines, fetchLogs, hydrateHistory, normalizedPageSize, resetViewerState, schedulePolling, setError])

  const refresh = useCallback(() => {
    if (enabled) void startGeneration(false)
  }, [enabled, startGeneration])

  const stopPolling = useCallback(() => {
    cancelFollow()
  }, [cancelFollow])

  const handleViewportScroll = useCallback(() => {
    const viewport = viewportRef.current
    if (!viewport) return
    if (programmaticAutoScrollRef.current && autoScrollRef.current) return
    const viewportSize = { width: viewport.clientWidth, height: viewport.clientHeight }
    const previousViewportSize = viewportSizeRef.current
    viewportSizeRef.current = viewportSize
    if (
      autoScrollRef.current &&
      previousViewportSize &&
      (previousViewportSize.width !== viewportSize.width || previousViewportSize.height !== viewportSize.height)
    ) {
      scheduleAutoScrollSettlement()
      return
    }
    const nearBottom = viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight < VIEWPORT_BOTTOM_THRESHOLD_PX
    setAutoScrollState(nearBottom)
  }, [scheduleAutoScrollSettlement, setAutoScrollState])

  const jumpToLatest = useCallback(() => {
    setAutoScrollState(true)
    const viewport = viewportRef.current
    if (viewport) viewport.scrollTop = viewport.scrollHeight
  }, [setAutoScrollState])

  useEffect(() => {
    const previous = previousAutoRefreshRef.current
    previousAutoRefreshRef.current = pollingEnabled
    autoRefreshRef.current = pollingEnabled
    if (
      !enabled ||
      !initializedRef.current ||
      initializationGenerationRef.current === generationRef.current
    ) {
      return
    }
    if (previous === pollingEnabled) return
    if (!pollingEnabled) {
      cancelFollow()
      setPhase((current) => (current === "error" ? current : "ready"))
      return
    }
    void runFollowRef.current(generationRef.current)
  }, [cancelFollow, enabled, pollingEnabled])

  useEffect(() => {
    if (!enabled || typeof ResizeObserver === "undefined") return
    const viewport = viewportRef.current
    if (!viewport) return

    viewportSizeRef.current = { width: viewport.clientWidth, height: viewport.clientHeight }
    const observer = new ResizeObserver(([entry]) => {
      const nextSize = {
        width: Math.round(entry?.contentRect.width ?? viewport.clientWidth),
        height: Math.round(entry?.contentRect.height ?? viewport.clientHeight),
      }
      const previousSize = viewportSizeRef.current
      viewportSizeRef.current = nextSize
      if (
        autoScrollRef.current &&
        previousSize &&
        (previousSize.width !== nextSize.width || previousSize.height !== nextSize.height)
      ) {
        scheduleAutoScrollSettlement()
      }
    })
    observer.observe(viewport)
    return () => observer.disconnect()
  }, [enabled, scheduleAutoScrollSettlement])

  useEffect(() => {
    if (!enabled) {
      cancelGeneration()
      initializedRef.current = false
      resetViewerState()
      return
    }
    const sourceChanged = sourceRef.current !== fetchLogs
    sourceRef.current = fetchLogs
    if (!initializedRef.current || sourceChanged) {
      initializedRef.current = true
      previousWindowSizeRef.current = normalizedWindowSize
      void startGeneration(false)
    }
  }, [cancelGeneration, enabled, fetchLogs, normalizedWindowSize, resetViewerState, startGeneration])

  useEffect(() => {
    const previous = previousWindowSizeRef.current
    if (previous === normalizedWindowSize) return
    previousWindowSizeRef.current = normalizedWindowSize
    if (!enabled || !initializedRef.current) return
    if (normalizedWindowSize < previous) {
      const retainedLines = linesRef.current.slice(-normalizedWindowSize)
      const trimmed = retainedLines.length < linesRef.current.length
      commitLines(retainedLines)
      if (trimmed) setTrimmedBefore(true)
      olderCursorRef.current = ""
      setHasOlder(false)
      const wasHydrating = historyHydrationGenerationRef.current === generationRef.current
      if (wasHydrating) {
        // A smaller window no longer needs this older cursor. Abort it before its
        // response can reintroduce discarded history metadata, then follow from
        // the latest-page cursor without rebuilding the window over the network.
        cancelHistoryHydration()
        if (initializationGenerationRef.current === generationRef.current) {
          initializationGenerationRef.current = null
        }
      }
      // Follow responses are generation-scoped but can still be in flight while
      // this local window changes. Restart from the retained follow cursor so a
      // late response cannot commit metadata after the smaller window is active.
      cancelFollow()
      if (autoRefreshRef.current) {
        void runFollowRef.current(generationRef.current)
      } else if (wasHydrating) {
        setPhase("ready")
      }
      return
    }
    void startGeneration(true)
  }, [cancelFollow, cancelHistoryHydration, commitLines, enabled, normalizedWindowSize, startGeneration])

  useLayoutEffect(() => {
    if (!autoScroll) return
    scheduleAutoScrollSettlement()
  }, [autoScroll, lines, scheduleAutoScrollSettlement])

  useEffect(() => () => {
    // React development Strict Mode replays effects. A canceled first request
    // must leave the next setup eligible to start a fresh generation.
    initializedRef.current = false
    cancelGeneration()
  }, [cancelGeneration])

  return {
    lines,
    phase,
    errorCode,
    errorMessage,
    autoScroll,
    setAutoScroll: setAutoScrollState,
    hasOlder,
    hasNewer,
    caughtUp,
    gap,
    gapReason,
    trimmedBefore,
    trimmedAfter,
    viewportRef,
    stopPolling,
    refresh,
    handleViewportScroll,
    jumpToLatest,
  }
}

function hasLogQueryErrorShape(error: unknown): error is Error & { code: string } {
  return error instanceof Error && "code" in error && typeof (error as { code?: unknown }).code === "string"
}

function isRecoverableLogQueryError(error: unknown) {
  if (!hasLogQueryErrorShape(error)) return true
  const status = (error as Error & { status?: unknown }).status
  return typeof status !== "number" || status >= 500 || error.code === "network_error" || error.code === "query_timeout"
}

function isAbortError(error: unknown) {
  if (error instanceof DOMException) return error.name === "AbortError"
  if (!(error instanceof Error)) return false
  const maybeAxiosCancel = error as Error & { code?: unknown }
  return error.name === "AbortError" || error.name === "CanceledError" || maybeAxiosCancel.code === "ERR_CANCELED"
}
