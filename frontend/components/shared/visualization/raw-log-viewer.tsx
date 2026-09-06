"use client"

import { useDeferredValue, useEffect, useMemo, useRef } from "react"
import AnsiToHtml from "ansi-to-html"

import { cn } from "@/lib/utils"
import { getTerminalLogLevelColor } from "@/lib/log-level"

import { TERMINAL_LOG_VIEWPORT_CLASS } from "./terminal-log-surface"

export type RawLogLevel = "all" | "DEBUG" | "INFO" | "WARN" | "ERROR"

interface RawLogViewerProps {
  content: string
  className?: string
  searchQuery?: string
  logLevel?: RawLogLevel
  topRightAction?: React.ReactNode
}

const ansiConverter = new AnsiToHtml({
  fg: "var(--terminal-log-foreground)",
  bg: "var(--terminal-log-background)",
  newline: false,
  escapeXML: true,
  colors: {
    0: "var(--terminal-log-background)",
    1: "var(--terminal-log-error)",
    2: "var(--terminal-log-info)",
    3: "var(--terminal-log-warning)",
    4: "var(--info)",
    5: "var(--info)",
    6: "var(--terminal-log-debug)",
    7: "var(--terminal-log-foreground)",
    8: "var(--terminal-log-muted)",
    9: "var(--terminal-log-error)",
    10: "var(--terminal-log-info)",
    11: "var(--terminal-log-warning)",
    12: "var(--info)",
    13: "var(--info)",
    14: "var(--terminal-log-debug)",
    15: "var(--logo-background)",
  },
})

function hasAnsiCodes(text: string): boolean {
  return /\x1b\[|\u001b\[/.test(text)
}

function colorizeLogContent(content: string): string {
  const logLineRegex = /^(\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d{3})?\]) (\[(DEBUG|INFO|WARNING|WARN|ERROR|CRITICAL)\]) (.*)$/i

  return content
    .split("\n")
    .map((line) => {
      const match = line.match(logLineRegex)
      if (!match) {
        return ansiConverter.toHtml(line)
      }

      const [, timestamp, levelBracket, level, rest] = match
      const levelUpper = level.toUpperCase()
      const color = getTerminalLogLevelColor(levelUpper)
      const escapedTimestamp = ansiConverter.toHtml(timestamp)
      const escapedLevelBracket = ansiConverter.toHtml(levelBracket)
      const escapedRest = ansiConverter.toHtml(rest)

      return `<span style="color:var(--terminal-log-muted)">${escapedTimestamp}</span> <span style="color:${color};font-weight:${levelUpper === "CRITICAL" ? "bold" : "normal"}">${escapedLevelBracket}</span> ${escapedRest}`
    })
    .join("\n")
}

function highlightSearch(html: string, query: string): string {
  if (!query.trim()) return html

  const escapedQueryForHtml = ansiConverter.toHtml(query)
  const escapedQuery = escapedQueryForHtml.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
  const regex = new RegExp(`(${escapedQuery})`, "giu")

  return html.replace(/(<[^>]+>)|([^<]+)/g, (match, tag, text) => {
    if (tag) return tag
    if (!text) return match

    return text.replace(
      regex,
      '<mark style="background:var(--terminal-log-highlight);color:var(--terminal-log-highlight-foreground);border-radius:2px;padding:0 2px">$1</mark>'
    )
  })
}

const LOG_LEVEL_PATTERNS = [
  /^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d{3})?\] \[(DEBUG|INFO|WARNING|WARN|ERROR|CRITICAL)\]/i,
  /^[\d:.]+\s+\|\s+(DEBUG|INFO|WARNING|WARN|ERROR|CRITICAL)\s+\|/i,
  /^(?:\[)?(DEBUG|INFO|WARNING|WARN|ERROR|CRITICAL)(?:\])?[:\s]/i,
  /^(DEBUG|INFO|WARNING|WARN|ERROR|CRITICAL)\s+-\s+/i,
]

const NEW_ENTRY_PATTERNS = [
  /^\[\d+\/\d+\]/,
  /^\[CONFIG\]/i,
  /^\[诊断\]/,
  /^={10,}$/,
  /^\[\d{4}-\d{2}-\d{2}/,
  /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}/,
  /^\/[\w/]+\.py:\d+:/,
]

function extractLogLevel(line: string): string | null {
  for (const pattern of LOG_LEVEL_PATTERNS) {
    const match = line.match(pattern)
    if (match) return match[1].toUpperCase()
  }
  return null
}

function isNewEntryStart(line: string): boolean {
  return NEW_ENTRY_PATTERNS.some((pattern) => pattern.test(line))
}

function normalizeLevel(level: string): string {
  const upper = level.toUpperCase()
  if (upper === "WARNING") return "WARN"
  if (upper === "CRITICAL") return "ERROR"
  return upper
}

function filterByLevel(content: string, level: RawLogLevel): string {
  if (level === "all") return content

  const targetLevel = normalizeLevel(level)
  const result: string[] = []
  let currentBlockVisible = false

  for (const line of content.split("\n")) {
    const extractedLevel = extractLogLevel(line)
    if (extractedLevel) {
      currentBlockVisible = normalizeLevel(extractedLevel) === targetLevel
    } else if (isNewEntryStart(line)) {
      currentBlockVisible = false
    }
    if (currentBlockVisible) result.push(line)
  }

  return result.join("\n")
}

export function RawLogViewer({
  content,
  className,
  searchQuery = "",
  logLevel = "all",
  topRightAction,
}: RawLogViewerProps) {
  const containerRef = useRef<HTMLPreElement>(null)
  const isAtBottomRef = useRef(true)
  const deferredQuery = useDeferredValue(searchQuery)

  const baseHtml = useMemo(() => {
    if (!content) return ""

    const filteredContent = filterByLevel(content, logLevel)
    return hasAnsiCodes(filteredContent)
      ? ansiConverter.toHtml(filteredContent)
      : colorizeLogContent(filteredContent)
  }, [content, logLevel])

  const htmlContent = useMemo(
    () => highlightSearch(baseHtml, deferredQuery),
    [baseHtml, deferredQuery]
  )

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    const handleScroll = () => {
      const { scrollTop, scrollHeight, clientHeight } = container
      isAtBottomRef.current = scrollHeight - scrollTop - clientHeight < 30
    }

    container.addEventListener("scroll", handleScroll, { passive: true })
    return () => container.removeEventListener("scroll", handleScroll)
  }, [])

  useEffect(() => {
    if (containerRef.current && isAtBottomRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight
    }
  }, [htmlContent])

  const viewer = (
    <pre
      ref={containerRef}
      data-slot="raw-log-viewer"
      className={cn(
        TERMINAL_LOG_VIEWPORT_CLASS,
        "m-0 w-full whitespace-pre-wrap break-all bg-[var(--terminal-log-background)] text-[var(--terminal-log-foreground)]",
        className
      )}
      dangerouslySetInnerHTML={{ __html: htmlContent }}
    />
  )

  if (!topRightAction) {
    return viewer
  }

  return (
    <div data-slot="raw-log-surface" className="relative h-full">
      {viewer}
      <div data-slot="raw-log-top-right-action" className="absolute right-3 top-3 z-10">
        {topRightAction}
      </div>
    </div>
  )
}
