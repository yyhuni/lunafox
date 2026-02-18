"use client"

import { useMemo, useRef, useEffect, useDeferredValue } from "react"
import AnsiToHtml from "ansi-to-html"
import type { LogLevel } from "./log-toolbar"

interface AnsiLogViewerProps {
  content: string
  className?: string
  searchQuery?: string
  logLevel?: LogLevel
}

// Log level color configuration
const LOG_LEVEL_COLORS: Record<string, string> = {
  DEBUG: "#4ec9b0",    // cyan
  INFO: "#6a9955",     // green
  WARNING: "#dcdcaa",  // yellow
  WARN: "#dcdcaa",     // yellow
  ERROR: "#f44747",    // red
  CRITICAL: "#f44747", // red (bold handled separately)
}

// Create an ANSI converter instance
const ansiConverter = new AnsiToHtml({
  fg: "#d4d4d4",
  bg: "#1e1e1e",
  newline: false,  // We handle line breaks ourselves
  escapeXML: true,
  colors: {
    0: "#1e1e1e",   // black
    1: "#f44747",   // red
    2: "#6a9955",   // green
    3: "#dcdcaa",   // yellow
    4: "#569cd6",   // blue
    5: "#c586c0",   // magenta
    6: "#4ec9b0",   // cyan
    7: "#d4d4d4",   // white
    8: "#808080",   // bright black
    9: "#f44747",   // bright red
    10: "#6a9955",  // bright green
    11: "#dcdcaa",  // bright yellow
    12: "#569cd6",  // bright blue
    13: "#c586c0",  // bright magenta
    14: "#4ec9b0",  // bright cyan
    15: "#ffffff",  // bright white
  },
})

// Detect whether content contains ANSI color codes
function hasAnsiCodes(text: string): boolean {
  // ANSI escape sequences usually start with ESC[ (\x1b[ or \u001b[)
  return /\x1b\[|\u001b\[/.test(text)
}

// Parse plain text log content and add color to log level
function colorizeLogContent(content: string): string {
  // Match log format:
  // 1) System log: [2026-01-10 09:51:52] [INFO] [apps.scan.xxx:123] ...
  // 2) Scan log: [09:50:37] [INFO] [subdomain_discovery] ...
  const logLineRegex = /^(\[(?:\d{4}-\d{2}-\d{2} )?\d{2}:\d{2}:\d{2}\]) (\[(DEBUG|INFO|WARNING|WARN|ERROR|CRITICAL)\]) (.*)$/i
  
  return content
    .split("\n")
    .map((line) => {
      const match = line.match(logLineRegex)
      
      if (match) {
        const [, timestamp, levelBracket, level, rest] = match
        const levelUpper = level.toUpperCase()
        const color = LOG_LEVEL_COLORS[levelUpper] || "#d4d4d4"
        // ansiConverter.toHtml already handles HTML escaping
        const escapedTimestamp = ansiConverter.toHtml(timestamp)
        const escapedLevelBracket = ansiConverter.toHtml(levelBracket)
        const escapedRest = ansiConverter.toHtml(rest)
        
        // The timestamp is gray, the log level is colored, and the rest are in default colors.
        return `<span style="color:#808080">${escapedTimestamp}</span> <span style="color:${color};font-weight:${levelUpper === "CRITICAL" ? "bold" : "normal"}">${escapedLevelBracket}</span> ${escapedRest}`
      }
      
      // Non-standard lines are also HTML-escaped.
      return ansiConverter.toHtml(line)
    })
    .join("\n")
}

// Highlight search keywords
function highlightSearch(html: string, query: string): string {
  if (!query.trim()) return html

  // `ansi-to-html` will convert non-ASCII characters (such as Chinese) into entities when `escapeXML: true` is set:
  // For example, a non-ASCII word can become an entity sequence like "&#x4E2D;&#x6587;".
  // Therefore, the same escaping rules need to be used here to generate a matching search string.
  const escapedQueryForHtml = ansiConverter.toHtml(query)

  // Escape regular special characters
  const escapedQuery = escapedQueryForHtml.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
  const regex = new RegExp(`(${escapedQuery})`, "giu")

  // Highlight keywords in text outside tags
  return html.replace(/(<[^>]+>)|([^<]+)/g, (match, tag, text) => {
    if (tag) return tag
    if (text) {
      return text.replace(
        regex,
        '<mark style="background:#fbbf24;color:#1e1e1e;border-radius:2px;padding:0 2px">$1</mark>'
      )
    }
    return match
  })
}

// Log-level extraction patterns for multiple log formats.
const LOG_LEVEL_PATTERNS = [
  // Standard format: [2026-01-07 12:00:00] [INFO]
  /^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\] \[(DEBUG|INFO|WARNING|WARN|ERROR|CRITICAL)\]/i,
  // Scan log format: [09:50:37] [INFO] [stage]
  /^\[\d{2}:\d{2}:\d{2}\] \[(DEBUG|INFO|WARNING|WARN|ERROR|CRITICAL)\]/i,
  // Prefect format: 12:01:50.419 | WARNING | prefect
  /^[\d:.]+\s+\|\s+(DEBUG|INFO|WARNING|WARN|ERROR|CRITICAL)\s+\|/i,
  // Simple format: [INFO] message or INFO: message
  /^(?:\[)?(DEBUG|INFO|WARNING|WARN|ERROR|CRITICAL)(?:\])?[:\s]/i,
  // Python logging format: INFO - message
  /^(DEBUG|INFO|WARNING|WARN|ERROR|CRITICAL)\s+-\s+/i,
]

// New-entry patterns (no level, but mark the start of a new entry).
const NEW_ENTRY_PATTERNS = [
  /^\[\d+\/\d+\]/, // [1/4], [2/4], etc. step markers
  /^\[CONFIG\]/i, // [CONFIG] metadata line
  /^\[诊断\]/, // Diagnostics label line
  /^={10,}$/, // ============ separator line
  /^\[\d{4}-\d{2}-\d{2}/, // Timestamp prefix, e.g. [2026-01-07...
  /^\d{2}:\d{2}:\d{2}/, // Time prefix, e.g. 12:01:50...
  /^\/[\w/]+\.py:\d+:/, // Python path prefix, e.g. /path/file.py:123:
]

// Extract log level from a line.
function extractLogLevel(line: string): string | null {
  for (const pattern of LOG_LEVEL_PATTERNS) {
    const match = line.match(pattern)
    if (match) {
      return match[1].toUpperCase()
    }
  }
  return null
}

// Check whether the line starts a new log entry (without a level).
function isNewEntryStart(line: string): boolean {
  return NEW_ENTRY_PATTERNS.some((pattern) => pattern.test(line))
}

// Normalize level labels.
function normalizeLevel(l: string): string {
  const upper = l.toUpperCase()
  if (upper === "WARNING") return "WARN"
  if (upper === "CRITICAL") return "ERROR"
  return upper
}

// Filter log lines by level.
// For multi-line logs, non-standard lines inherit the previous standard line's level.
function filterByLevel(content: string, level: LogLevel): string {
  if (level === "all") return content
  
  const targetLevel = normalizeLevel(level)
  const lines = content.split("\n")
  const result: string[] = []
  // Hidden by default until a line matching the target level is encountered.
  let currentBlockVisible = false
  
  for (const line of lines) {
    const extractedLevel = extractLogLevel(line)
    if (extractedLevel) {
      // This is a new log entry; evaluate visibility by exact level match.
      const lineLevel = normalizeLevel(extractedLevel)
      currentBlockVisible = lineLevel === targetLevel
    } else if (isNewEntryStart(line)) {
      // New entry without explicit level; hide by default.
      currentBlockVisible = false
    }
    // Non-standard lines follow the previous entry's visibility.
    if (currentBlockVisible) {
      result.push(line)
    }
  }
  
  return result.join("\n")
}

export function AnsiLogViewer({ content, className, searchQuery = "", logLevel = "all" }: AnsiLogViewerProps) {
  const containerRef = useRef<HTMLPreElement>(null)
  const isAtBottomRef = useRef(true)  // Track whether the user is near the bottom.
  const deferredQuery = useDeferredValue(searchQuery)

  // Parse logs and apply colors.
  // Supports two modes: ANSI color codes and plain-text level parsing.
  const baseHtml = useMemo(() => {
    if (!content) return ""
    
    // Apply level filtering first.
    const filteredContent = filterByLevel(content, logLevel)
    
    let result: string
    // If ANSI escape codes exist, convert directly.
    if (hasAnsiCodes(filteredContent)) {
      result = ansiConverter.toHtml(filteredContent)
    } else {
      // Otherwise, parse log levels and apply color styles.
      result = colorizeLogContent(filteredContent)
    }
    
    return result
  }, [content, logLevel])

  // Apply search highlighting (deferred query avoids typing lag).
  const htmlContent = useMemo(
    () => highlightSearch(baseHtml, deferredQuery),
    [baseHtml, deferredQuery]
  )

  // Track scroll position and detect whether the user is near the bottom.
  useEffect(() => {
    const container = containerRef.current
    if (!container) return
    
    const handleScroll = () => {
      const { scrollTop, scrollHeight, clientHeight } = container
      // Treat within 30px as near-bottom.
      isAtBottomRef.current = scrollHeight - scrollTop - clientHeight < 30
    }
    
    container.addEventListener('scroll', handleScroll, { passive: true })
    return () => container.removeEventListener('scroll', handleScroll)
  }, [])

  // Auto-scroll only when the user is already near the bottom.
  useEffect(() => {
    if (containerRef.current && isAtBottomRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight
    }
  }, [htmlContent])

  return (
    <pre
      ref={containerRef}
      className={className}
      style={{
        height: "100%",
        width: "100%",
        margin: 0,
        padding: "12px",
        overflow: "auto",
        backgroundColor: "#1e1e1e",
        color: "#d4d4d4",
        fontSize: "12px",
        fontFamily: 'Menlo, Monaco, "Courier New", monospace',
        lineHeight: 1.5,
        whiteSpace: "pre-wrap",
        wordBreak: "break-all",
      }}
      dangerouslySetInnerHTML={{ __html: htmlContent }}
    />
  )
}
