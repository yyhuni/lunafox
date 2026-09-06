"use client"

import { useLayoutEffect, useMemo, useRef, useState, type CSSProperties, type RefObject } from "react"
import { useVirtualizer } from "@tanstack/react-virtual"
import { cn } from "@/lib/utils"
import { getTerminalLogLevelColor } from "@/lib/log-level"
import { formatLogTimestamp } from "@/lib/log-time"

export const MAX_STRUCTURED_LOG_RENDER_LINES = 5000

const STRUCTURED_LOG_VIRTUALIZATION_THRESHOLD = 200
const STRUCTURED_LOG_ESTIMATED_ROW_HEIGHT_PX = 20
const STRUCTURED_LOG_OVERSCAN_ROWS = 12

export type StructuredLogLevel = "debug" | "info" | "warn" | "error" | "fatal"
export type StructuredLogLevelFilter = "all" | StructuredLogLevel

const LEVEL_LABEL: Record<StructuredLogLevel, string> = {
  debug: "DEBUG",
  info: "INFO",
  warn: "WARN",
  error: "ERROR",
  fatal: "FATAL",
}

export interface StructuredLogLineItem {
  id: string
  ts: string
  stream: string
  line: string
  truncated: boolean
}

export interface ParsedLogLine {
  level: StructuredLogLevel | null
  msg: string
  caller: string
  extras: Array<{ key: string; value: string }>
}

export function normalizeLogLevel(raw: string): StructuredLogLevel | null {
  const lower = raw.toLowerCase().trim()
  if (lower === "debug" || lower === "dbg") return "debug"
  if (lower === "info" || lower === "information") return "info"
  if (lower === "warn" || lower === "warning") return "warn"
  if (lower === "error" || lower === "err") return "error"
  if (lower === "fatal" || lower === "critical" || lower === "panic") return "fatal"
  return null
}

export function parseLogLine(raw: string): ParsedLogLine | null {
  if (!raw || raw[0] !== "{") return null
  try {
    const obj = JSON.parse(raw)
    if (typeof obj !== "object" || obj === null || Array.isArray(obj)) return null

    const level = typeof obj.level === "string" ? normalizeLogLevel(obj.level) : null
    const msg = typeof obj.msg === "string" ? obj.msg : ""
    const caller = typeof obj.caller === "string" ? obj.caller : ""

    const reserved = new Set(["level", "msg", "caller", "ts", "time", "timestamp"])
    const extras: Array<{ key: string; value: string }> = []
    for (const [key, value] of Object.entries(obj)) {
      if (reserved.has(key)) continue
      if (value === null || value === undefined) continue
      extras.push({ key, value: typeof value === "object" ? JSON.stringify(value) : String(value) })
    }

    return { level, msg, caller, extras }
  } catch {
    return null
  }
}

export function filterStructuredLogLines<TLine extends StructuredLogLineItem>(
  lines: TLine[],
  searchTerm: string,
  levelFilter: StructuredLogLevelFilter,
) {
  const term = searchTerm.trim().toLowerCase()
  return lines.filter((line) => {
    if (term && !line.line.toLowerCase().includes(term)) return false
    if (levelFilter === "all") return true

    const parsed = parseLogLine(line.line)
    if (!parsed?.level) return false
    return parsed.level === levelFilter
  })
}

export const formatLogTime = formatLogTimestamp

export function formatStructuredLogLine(
  line: StructuredLogLineItem,
  truncatedLabel = "truncated",
): string {
  const parsed = parseLogLine(line.line)
  const content = parsed ? formatStructuredLogContent(parsed) : line.line || " "
  const truncation = line.truncated ? ` [${truncatedLabel}]` : ""

  return `[${formatLogTime(line.ts)}] ${content}${truncation}`
}

export function isErrorStream(stream: string): boolean {
  return stream.toLowerCase() === "stderr"
}

export function StructuredLogViewer({
  viewportRef,
  lines,
  empty,
  className,
  truncatedLabel = "truncated",
}: {
  viewportRef: RefObject<HTMLDivElement | null>
  lines: StructuredLogLineItem[]
  empty: React.ReactNode
  className?: string
  truncatedLabel?: string
}) {
  const viewerRef = useRef<HTMLPreElement | null>(null)
  const [scrollMargin, setScrollMargin] = useState(0)
  const virtualized = lines.length > STRUCTURED_LOG_VIRTUALIZATION_THRESHOLD
  const virtualizer = useVirtualizer<HTMLDivElement, HTMLSpanElement>({
    count: lines.length,
    getScrollElement: () => viewportRef.current,
    estimateSize: () => STRUCTURED_LOG_ESTIMATED_ROW_HEIGHT_PX,
    getItemKey: (index) => lines[index]?.id ?? index,
    overscan: STRUCTURED_LOG_OVERSCAN_ROWS,
    scrollMargin,
    anchorTo: "end",
    followOnAppend: true,
    scrollEndThreshold: 24,
    enabled: virtualized,
    // Measurement notifications can run from a React layout effect. Let React
    // schedule the rerender instead of attempting a nested synchronous flush.
    useFlushSync: false,
  })

  // Gap and trimming banners live above this component and may move it while
  // the lines reference stays stable, so recompute the margin after each render.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useLayoutEffect(() => {
    if (!virtualized) return
    const viewer = viewerRef.current
    const viewport = viewportRef.current
    if (!viewer || !viewport) return

    const nextScrollMargin =
      viewer.getBoundingClientRect().top - viewport.getBoundingClientRect().top + viewport.scrollTop
    setScrollMargin((current) => current === nextScrollMargin ? current : nextScrollMargin)
  })

  if (lines.length === 0) {
    return <div className="text-muted-foreground">{empty}</div>
  }

  if (virtualized) {
    return (
      <pre
        ref={viewerRef}
        data-slot="structured-log-viewer"
        data-total-lines={lines.length}
        className={cn("relative m-0 min-w-full whitespace-pre-wrap break-all", className)}
        style={{ height: virtualizer.getTotalSize() }}
      >
        {virtualizer.getVirtualItems().map((virtualRow) => (
          <StructuredLogLine
            key={virtualRow.key}
            ref={virtualizer.measureElement}
            dataIndex={virtualRow.index}
            line={lines[virtualRow.index]}
            truncatedLabel={truncatedLabel}
            trailingNewline={false}
            style={{
              position: "absolute",
              top: 0,
              left: 0,
              width: "100%",
              transform: `translateY(${virtualRow.start - scrollMargin}px)`,
            }}
          />
        ))}
      </pre>
    )
  }

  return (
    <pre
      ref={viewerRef}
      data-slot="structured-log-viewer"
      data-total-lines={lines.length}
      className={cn("m-0 min-w-full whitespace-pre-wrap break-all", className)}
    >
      {lines.map((line, index) => (
        <StructuredLogLine
          key={line.id}
          line={line}
          truncatedLabel={truncatedLabel}
          trailingNewline={index < lines.length - 1}
        />
      ))}
    </pre>
  )
}

function StructuredLogLine({
  ref,
  dataIndex,
  line,
  truncatedLabel,
  trailingNewline,
  style,
}: {
  ref?: (node: HTMLSpanElement | null) => void
  dataIndex?: number
  line: StructuredLogLineItem
  truncatedLabel: string
  trailingNewline: boolean
  style?: CSSProperties
}) {
  const parsed = useMemo(() => parseLogLine(line.line), [line.line])
  const isError = isErrorStream(line.stream)

  return (
    <>
      <span ref={ref} data-index={dataIndex} data-slot="structured-log-line" style={style}>
        <span style={{ color: "var(--terminal-log-muted)" }}>[{formatLogTime(line.ts)}]</span>{" "}
        {parsed ? (
          <StructuredContent parsed={parsed} />
        ) : (
          <span style={{ color: isError ? "var(--terminal-log-error)" : "var(--terminal-log-foreground)" }}>
            {line.line || " "}
          </span>
        )}
        {line.truncated && (
          <>
            {" "}
            <span style={{ color: "var(--terminal-log-warning)" }}>[{truncatedLabel}]</span>
          </>
        )}
      </span>
      {trailingNewline ? "\n" : null}
    </>
  )
}

function StructuredContent({ parsed }: { parsed: ParsedLogLine }) {
  const isErrorLevel = parsed.level === "error" || parsed.level === "fatal"
  const tokens = [
    parsed.level ? (
      <span
        key="level"
        style={{ color: getTerminalLogLevelColor(parsed.level) }}
      >
        [{LEVEL_LABEL[parsed.level]}]
      </span>
    ) : null,
    parsed.msg ? (
      <span key="msg" style={{ color: isErrorLevel ? "var(--terminal-log-error)" : "var(--terminal-log-foreground)" }}>
        {parsed.msg}
      </span>
    ) : null,
    parsed.caller ? (
      <span key="caller" style={{ color: "var(--terminal-log-muted)" }}>{parsed.caller}</span>
    ) : null,
    ...parsed.extras.map(({ key, value }) => (
      <span key={key}>
        <span style={{ color: "var(--terminal-log-muted)" }}>{key}=</span>
        <span style={{ color: "var(--terminal-log-foreground)" }}>{value}</span>
      </span>
    )),
  ].filter(Boolean)

  return (
    <>
      {tokens.map((token, index) => (
        <span key={index}>
          {index > 0 ? " " : null}
          {token}
        </span>
      ))}
    </>
  )
}

function formatStructuredLogContent(parsed: ParsedLogLine): string {
  return [
    parsed.level ? `[${LEVEL_LABEL[parsed.level]}]` : "",
    parsed.msg,
    parsed.caller,
    ...parsed.extras.map(({ key, value }) => `${key}=${value}`),
  ].filter(Boolean).join(" ")
}
