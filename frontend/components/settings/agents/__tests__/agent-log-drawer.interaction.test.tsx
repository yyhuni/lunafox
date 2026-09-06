import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import type { ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import type { Agent } from "@/types/agent.types"

const logViewerMocks = vi.hoisted(() => ({
  handleViewportScroll: vi.fn(),
  jumpToLatest: vi.fn(),
  refresh: vi.fn(),
  stopPolling: vi.fn(),
  useAgentLogs: vi.fn(),
}))

vi.mock("@/hooks/use-agent-logs", () => ({
  useAgentLogs: logViewerMocks.useAgentLogs,
}))

vi.mock("@/components/ui/sheet", () => ({
  Sheet: ({ children, onOpenChange }: { children: ReactNode; onOpenChange?: (open: boolean) => void }) => (
    <div>
      <button type="button" onClick={() => onOpenChange?.(false)}>Close drawer</button>
      {children}
    </div>
  ),
  SheetContent: ({ children }: { children: ReactNode }) => <div data-slot="sheet-content">{children}</div>,
  SheetHeader: ({ children }: { children: ReactNode }) => <header>{children}</header>,
  SheetTitle: ({ children }: { children: ReactNode }) => <h2>{children}</h2>,
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
  TerminalLogCopyAllButton: ({ value, copyLabel }: { value: string; copyLabel: string }) => (
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
import { AgentLogDrawer } from "../agent-log-drawer"

const LOG_WINDOW_OPTIONS = [100, 200, 500, 1000, 2000, 5000]

const agentA: Agent = {
  id: 1,
  name: "Agent A",
  status: "online",
  connectionIp: "10.0.0.1",
  maxTasks: 2,
  cpuThreshold: 90,
  memThreshold: 90,
  diskThreshold: 90,
  health: { state: "healthy" },
  createdAt: "2026-07-25T00:00:00.000Z",
}

const agentB: Agent = {
  ...agentA,
  id: 2,
  name: "Agent B",
}

function createViewerState() {
  return {
    autoScroll: true,
    caughtUp: true,
    errorCode: null,
    errorMessage: "",
    gap: false,
    gapReason: "",
    handleViewportScroll: logViewerMocks.handleViewportScroll,
    hasNewer: false,
    jumpToLatest: logViewerMocks.jumpToLatest,
    lines: [
      { id: "one", line: "first", stream: "stdout", truncated: false, ts: "2026-07-25T00:00:00.000Z" },
      { id: "two", line: "second", stream: "stdout", truncated: false, ts: "2026-07-25T00:00:01.000Z" },
    ],
    phase: "following" as const,
    refresh: logViewerMocks.refresh,
    stopPolling: logViewerMocks.stopPolling,
    trimmedAfter: false,
    trimmedBefore: false,
    viewportRef: { current: null },
  }
}

describe("AgentLogDrawer line-window session interactions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    logViewerMocks.useAgentLogs.mockReturnValue(createViewerState())
  })

  it("starts at 100, displays the actual count, and resets the session for an Agent switch or close", async () => {
    const onOpenChange = vi.fn()
    const { rerender } = render(
      <AgentLogDrawer open onOpenChange={onOpenChange} agentNode={agentA} />,
    )

    expect(logViewerMocks.useAgentLogs).toHaveBeenLastCalledWith(expect.objectContaining({
      agentNode: agentA,
      open: true,
      pollingEnabled: true,
      windowSize: 100,
    }))
    expect(screen.getByText("2 / 100 logs.toolbar.linesUnit")).toBeInTheDocument()
    expect(Array.from((screen.getByRole("combobox", { name: "logs.toolbar.lineWindow" }) as HTMLSelectElement).options)
      .map((option) => Number(option.value))).toEqual(LOG_WINDOW_OPTIONS)

    fireEvent.change(screen.getByRole("combobox", { name: "logs.toolbar.lineWindow" }), {
      target: { value: "5000" },
    })

    await waitFor(() => {
      expect(logViewerMocks.useAgentLogs).toHaveBeenLastCalledWith(expect.objectContaining({
        windowSize: 5000,
      }))
    })
    expect(screen.getByText("2 / 5000 logs.toolbar.linesUnit")).toBeInTheDocument()

    rerender(<AgentLogDrawer open onOpenChange={onOpenChange} agentNode={agentB} />)

    await waitFor(() => {
      expect(logViewerMocks.useAgentLogs).toHaveBeenLastCalledWith(expect.objectContaining({
        agentNode: agentB,
        windowSize: 100,
      }))
    })

    fireEvent.change(screen.getByRole("combobox", { name: "logs.toolbar.lineWindow" }), {
      target: { value: "2000" },
    })
    fireEvent.click(screen.getByRole("switch"))
    await waitFor(() => {
      expect(logViewerMocks.useAgentLogs).toHaveBeenLastCalledWith(expect.objectContaining({
        pollingEnabled: false,
        windowSize: 2000,
      }))
    })
    fireEvent.click(screen.getByRole("button", { name: "Close drawer" }))

    await waitFor(() => {
      expect(logViewerMocks.useAgentLogs).toHaveBeenLastCalledWith(expect.objectContaining({
        pollingEnabled: true,
        windowSize: 100,
      }))
    })
    expect(logViewerMocks.stopPolling).toHaveBeenCalledTimes(1)
    expect(onOpenChange).toHaveBeenLastCalledWith(false)
  })

  it("keeps the window session independent from the auto-refresh toggle", async () => {
    render(<AgentLogDrawer open onOpenChange={vi.fn()} agentNode={agentA} />)

    fireEvent.click(screen.getByRole("switch"))

    await waitFor(() => {
      expect(logViewerMocks.useAgentLogs).toHaveBeenLastCalledWith(expect.objectContaining({
        pollingEnabled: false,
        windowSize: 100,
      }))
    })
  })

  it("keeps a history hydration error visible while live follow is active", () => {
    logViewerMocks.useAgentLogs.mockReturnValue({
      ...createViewerState(),
      errorMessage: "older history could not be loaded",
      phase: "following",
    })

    render(<AgentLogDrawer open onOpenChange={vi.fn()} agentNode={agentA} />)

    expect(screen.getByText("older history could not be loaded")).toBeInTheDocument()
  })

  it("copies formatted text only for the log lines currently visible after level and search filters", async () => {
    const errorDatabaseLine = JSON.stringify({ level: "error", msg: "database query failed" })
    const warnDatabaseLine = JSON.stringify({ level: "warn", msg: "database query is slow" })
    const errorTimeoutLine = JSON.stringify({ level: "error", msg: "request timeout" })
    const lines = [
      { id: "error-database", line: errorDatabaseLine, stream: "stderr", truncated: false, ts: "2026-07-25T00:00:00.000Z" },
      { id: "warn-database", line: warnDatabaseLine, stream: "stdout", truncated: false, ts: "2026-07-25T00:00:01.000Z" },
      { id: "error-timeout", line: errorTimeoutLine, stream: "stderr", truncated: false, ts: "2026-07-25T00:00:02.000Z" },
    ]
    logViewerMocks.useAgentLogs.mockReturnValue({ ...createViewerState(), lines })

    render(<AgentLogDrawer open onOpenChange={vi.fn()} agentNode={agentA} />)

    const copyButton = screen.getByRole("button", { name: "logs.copyVisible" })
    expect(copyButton).toHaveAttribute("data-copy-value", lines.map((line) => formatStructuredLogLine(line)).join("\n"))

    fireEvent.click(screen.getByRole("button", { name: "ERR" }))
    await waitFor(() => {
      expect(copyButton).toHaveAttribute("data-copy-value", [lines[0], lines[2]].map((line) => formatStructuredLogLine(line)).join("\n"))
    })

    fireEvent.change(screen.getByRole("textbox", { name: "search logs" }), { target: { value: "timeout" } })
    await waitFor(() => {
      expect(copyButton).toHaveAttribute("data-copy-value", formatStructuredLogLine(lines[2]))
    })
  })
})
