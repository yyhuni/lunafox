import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import type { ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"

const systemLogMocks = vi.hoisted(() => ({
  handleViewportScroll: vi.fn(),
  jumpToLatest: vi.fn(),
  refresh: vi.fn(),
  useSystemLogs: vi.fn(),
}))

vi.mock("@/hooks/use-system-logs", () => ({
  useSystemLogs: systemLogMocks.useSystemLogs,
}))

vi.mock("@/components/common/page-header", () => ({
  PageHeader: ({ title }: { title: string }) => <h1>{title}</h1>,
}))

vi.mock("@/components/ui/switch", () => ({
  Switch: ({ checked, id, onCheckedChange }: {
    checked?: boolean
    id?: string
    onCheckedChange?: (checked: boolean) => void
  }) => (
    <button
      id={id}
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={() => onCheckedChange?.(!checked)}
    />
  ),
}))

vi.mock("@/components/shared/visualization/terminal-log-toolbar", () => ({
  TerminalLogToolbar: ({
    searchTerm,
    onSearchTermChange,
    levelFilter,
    onLevelFilterChange,
    lineWindow,
  }: {
    searchTerm: string
    onSearchTermChange: (value: string) => void
    levelFilter: string
    onLevelFilterChange: (value: "error") => void
    lineWindow?: {
      label: string
      onValueChange: (value: number) => void
      options: readonly number[]
      value: number
    }
  }) => (
    <>
      <input
        aria-label="search logs"
        value={searchTerm}
        onChange={(event) => onSearchTermChange(event.target.value)}
      />
      <button
        type="button"
        aria-pressed={levelFilter === "error"}
        onClick={() => onLevelFilterChange("error")}
      >
        ERR
      </button>
      {lineWindow ? (
        <select
          aria-label={lineWindow.label}
          value={lineWindow.value}
          onChange={(event) => lineWindow.onValueChange(Number(event.target.value))}
        >
          {lineWindow.options.map((option) => <option key={option} value={option}>{option}</option>)}
        </select>
      ) : null}
    </>
  ),
}))

vi.mock("@/components/shared/visualization/terminal-log-surface", () => ({
  LiveLogSurface: ({ children, footer, onScroll, topRightAction }: {
    children: ReactNode
    footer: ReactNode
    onScroll?: () => void
    topRightAction?: ReactNode
  }) => (
    <>
      {topRightAction}
      <div data-slot="live-log-terminal-body" onScroll={onScroll}>{children}</div>
      <footer>{footer}</footer>
    </>
  ),
}))

vi.mock("@/components/shared/visualization/terminal-log-copy-all-button", () => ({
  TerminalLogCopyAllButton: ({ value, copyLabel }: { value: string, copyLabel: string }) => (
    <button type="button" aria-label={copyLabel} data-copy-value={value}>Copy visible</button>
  ),
}))

vi.mock("@/components/shared/visualization/structured-log-viewer", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/components/shared/visualization/structured-log-viewer")>()

  return {
    ...actual,
    StructuredLogViewer: ({ lines }: { lines: Array<{ id: string; line: string }> }) => (
      <pre>{lines.map((line) => line.line).join("\n")}</pre>
    ),
  }
})

import { formatStructuredLogLine } from "@/components/shared/visualization/structured-log-viewer"
import { SystemLogsView } from "../system-logs-view"

const LOG_WINDOW_OPTIONS = [100, 200, 500, 1000, 2000, 5000]

function createViewerState() {
  return {
    autoScroll: true,
    caughtUp: true,
    errorCode: null,
    errorMessage: "",
    gap: false,
    gapReason: "",
    handleViewportScroll: systemLogMocks.handleViewportScroll,
    hasNewer: false,
    isLoading: false,
    jumpToLatest: systemLogMocks.jumpToLatest,
    lines: [
      { id: "one", line: "first", stream: "stdout", truncated: false, ts: "2026-07-25T00:00:00.000Z" },
      { id: "two", line: "second", stream: "stdout", truncated: false, ts: "2026-07-25T00:00:01.000Z" },
      { id: "three", line: "third", stream: "stdout", truncated: false, ts: "2026-07-25T00:00:02.000Z" },
    ],
    phase: "idle" as const,
    refresh: systemLogMocks.refresh,
    trimmedAfter: false,
    trimmedBefore: false,
    viewportRef: { current: null },
  }
}

describe("SystemLogsView line-window session interactions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    systemLogMocks.useSystemLogs.mockReturnValue(createViewerState())
  })

  it("starts with a 100-line window, reports actual rows, and resets on a fresh page entry", async () => {
    const initial = render(<SystemLogsView pageTitle="System logs" pageDescription="Operational logs" />)

    expect(systemLogMocks.useSystemLogs).toHaveBeenLastCalledWith({
      autoRefresh: true,
      windowSize: 100,
    })
    expect(screen.getByText("3 / 100 toolbar.linesUnit")).toBeInTheDocument()
    expect(Array.from((screen.getByRole("combobox", { name: "toolbar.lineWindow" }) as HTMLSelectElement).options)
      .map((option) => Number(option.value))).toEqual(LOG_WINDOW_OPTIONS)

    fireEvent.change(screen.getByRole("combobox", { name: "toolbar.lineWindow" }), {
      target: { value: "2000" },
    })

    await waitFor(() => {
      expect(systemLogMocks.useSystemLogs).toHaveBeenLastCalledWith({
        autoRefresh: true,
        windowSize: 2000,
      })
    })
    expect(screen.getByText("3 / 2000 toolbar.linesUnit")).toBeInTheDocument()

    initial.unmount()
    render(<SystemLogsView pageTitle="System logs" pageDescription="Operational logs" />)

    expect(systemLogMocks.useSystemLogs).toHaveBeenLastCalledWith({
      autoRefresh: true,
      windowSize: 100,
    })
  })

  it("forwards scroll handling and allows auto-refresh to be disabled then resumed without changing N", async () => {
    render(<SystemLogsView pageTitle="System logs" pageDescription="Operational logs" />)

    fireEvent.scroll(document.querySelector("[data-slot='live-log-terminal-body']")!)
    expect(systemLogMocks.handleViewportScroll).toHaveBeenCalledTimes(1)

    const autoRefresh = screen.getByRole("switch")
    fireEvent.click(autoRefresh)

    await waitFor(() => {
      expect(systemLogMocks.useSystemLogs).toHaveBeenLastCalledWith({
        autoRefresh: false,
        windowSize: 100,
      })
    })

    fireEvent.click(autoRefresh)

    await waitFor(() => {
      expect(systemLogMocks.useSystemLogs).toHaveBeenLastCalledWith({
        autoRefresh: true,
        windowSize: 100,
      })
    })
  })

  it("keeps a history hydration error visible while live follow is active", () => {
    systemLogMocks.useSystemLogs.mockReturnValue({
      ...createViewerState(),
      errorMessage: "older history could not be loaded",
      phase: "following",
    })

    render(<SystemLogsView pageTitle="System logs" pageDescription="Operational logs" />)

    expect(screen.getByText("older history could not be loaded")).toBeInTheDocument()
  })

  it("copies rendered text only for the log lines currently visible after level and search filters", async () => {
    const errorDatabaseLine = JSON.stringify({ level: "error", msg: "database query failed" })
    const warnDatabaseLine = JSON.stringify({ level: "warn", msg: "database query is slow" })
    const errorTimeoutLine = JSON.stringify({ level: "error", msg: "request timeout" })
    systemLogMocks.useSystemLogs.mockReturnValue({
      ...createViewerState(),
      lines: [
        { id: "error-database", line: errorDatabaseLine, stream: "stderr", truncated: false, ts: "2026-07-25T00:00:00.000Z" },
        { id: "warn-database", line: warnDatabaseLine, stream: "stdout", truncated: false, ts: "2026-07-25T00:00:01.000Z" },
        { id: "error-timeout", line: errorTimeoutLine, stream: "stderr", truncated: false, ts: "2026-07-25T00:00:02.000Z" },
      ],
    })

    render(<SystemLogsView pageTitle="System logs" pageDescription="Operational logs" />)

    const copyButton = screen.getByRole("button", { name: "logs.copyVisible" })
    expect(copyButton).toHaveAttribute("data-copy-value", [
      formatStructuredLogLine({ id: "error-database", line: errorDatabaseLine, stream: "stderr", truncated: false, ts: "2026-07-25T00:00:00.000Z" }),
      formatStructuredLogLine({ id: "warn-database", line: warnDatabaseLine, stream: "stdout", truncated: false, ts: "2026-07-25T00:00:01.000Z" }),
      formatStructuredLogLine({ id: "error-timeout", line: errorTimeoutLine, stream: "stderr", truncated: false, ts: "2026-07-25T00:00:02.000Z" }),
    ].join("\n"))

    fireEvent.click(screen.getByRole("button", { name: "ERR" }))
    await waitFor(() => {
      expect(copyButton).toHaveAttribute("data-copy-value", [
        formatStructuredLogLine({ id: "error-database", line: errorDatabaseLine, stream: "stderr", truncated: false, ts: "2026-07-25T00:00:00.000Z" }),
        formatStructuredLogLine({ id: "error-timeout", line: errorTimeoutLine, stream: "stderr", truncated: false, ts: "2026-07-25T00:00:02.000Z" }),
      ].join("\n"))
    })

    fireEvent.change(screen.getByRole("textbox", { name: "search logs" }), { target: { value: "timeout" } })
    await waitFor(() => {
      expect(copyButton).toHaveAttribute("data-copy-value", formatStructuredLogLine({
        id: "error-timeout",
        line: errorTimeoutLine,
        stream: "stderr",
        truncated: false,
        ts: "2026-07-25T00:00:02.000Z",
      }))
    })
  })
})
