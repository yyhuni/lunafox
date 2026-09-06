const TERMINAL_LOG_LEVEL_COLORS = {
  debug: "var(--terminal-log-debug)",
  info: "var(--terminal-log-info)",
  warn: "var(--terminal-log-warning)",
  error: "var(--terminal-log-error)",
} as const

type TerminalLogLevelTone = keyof typeof TERMINAL_LOG_LEVEL_COLORS

function normalizeTerminalLogLevel(raw: string): TerminalLogLevelTone | null {
  switch (raw.trim().toLowerCase()) {
    case "debug":
    case "dbg":
      return "debug"
    case "info":
    case "information":
      return "info"
    case "warn":
    case "warning":
      return "warn"
    case "error":
    case "err":
    case "critical":
    case "fatal":
    case "panic":
      return "error"
    default:
      return null
  }
}

export function getTerminalLogLevelColor(raw: string): string {
  const tone = normalizeTerminalLogLevel(raw)
  return tone ? TERMINAL_LOG_LEVEL_COLORS[tone] : "var(--terminal-log-foreground)"
}

export function getTerminalLogLevelBadgeStyle(raw: string) {
  const color = getTerminalLogLevelColor(raw)

  return {
    color,
    borderColor: `color-mix(in oklch, ${color} 55%, transparent)`,
    backgroundColor: `color-mix(in oklch, ${color} 10%, transparent)`,
  }
}
