import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-log-drawer.tsx"), "utf8")

describe("agent-log-drawer contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AgentLogDrawer")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared status helpers and semantic log surfaces", () => {
    expect(source).toContain("getStatusToneBgClass")
    expect(source).toContain("getStatusToneTextClass")
    expect(source).not.toContain("#0b1320")
    expect(source).not.toContain("#e5e7eb")
    expect(source).not.toContain("#6b7280")
    expect(source).not.toContain("#f87171")
    expect(source).not.toContain("#34d399")
    expect(source).not.toContain("#fecaca")
    expect(source).not.toContain("#d1fae5")
    expect(source).not.toContain("text-amber-300/90")
    expect(source).not.toContain("border-amber-400/40")
  })

  it("uses the shared current-connection display in the log context", () => {
    expect(source).toContain("getAgentConnectionIpDisplay")
    expect(source).not.toContain("agentNode.ipAddress")
    expect(source).not.toContain('t("unknownIp")')
  })

  it("uses the shared workbench drawer layout for the wide log workspace", () => {
    expect(source).toContain("workbenchLogDrawerContentClassName")
    expect(source).not.toContain("sideMotion")
    expect(source).not.toContain("detailDrawerBackdropClassName")
    expect(source).not.toContain("overlayClassName")
    expect(source).not.toContain('className="gap-0 p-0 sm:max-w-4xl w-full"')
  })

  it("does not use react-json-view-lite", () => {
    expect(source).not.toContain("react-json-view-lite")
    expect(source).not.toContain("JsonView")
    expect(source).not.toContain("collapseAllNested")
  })

  it("implements structured JSON log parsing", () => {
    expect(source).toContain("StructuredLogViewer")
    expect(source).toContain("filterStructuredLogLines")
    expect(source).toContain("formatStructuredLogLine")
    expect(source).toContain('from "@/components/shared/visualization/terminal-log-copy-all-button"')
    expect(source).toContain("const filteredLogText")
    expect(source).toContain("topRightAction={<TerminalLogCopyAllButton")
    expect(source).toContain("value={filteredLogText}")
    expect(source).toContain('copyLabel={t("logs.copyVisible")}')
    expect(source).toContain('toastId="agent-log-copy-all"')
    expect(source).not.toContain("function StructuredContent")
    expect(source).not.toContain("LEVEL_BADGE_STYLE")
    expect(source).toMatch(/<StructuredLogViewer\s+viewportRef=\{viewportRef\}/)
  })

  it("implements search and level filter controls", () => {
    expect(source).toContain("searchTerm")
    expect(source).toContain("levelFilter")
    expect(source).toContain("filteredLines")
    expect(source).toContain("TerminalLogToolbar")
    expect(source).toContain('from "@/components/shared/visualization/terminal-log-toolbar"')
  })

  it("keeps log search and level filters on the same compact control density", () => {
    expect(source).toContain("<TerminalLogToolbar searchTerm={searchTerm}")
    expect(source).not.toContain("h-7")
  })

  it("focuses the log viewport on open with a visible focus indicator instead of highlighting search", () => {
    expect(source).toContain('initialFocus={viewportRef}')
    expect(source).toContain("<LiveLogSurface")
    expect(source).toContain("focusable")
    expect(source).not.toContain('tabIndex={-1}')
    expect(source).not.toContain('focus-visible:ring-2 focus-visible:ring-ring/50')
    expect(source).not.toContain('focus:outline-none')
    expect(source).not.toContain("autoFocus")
  })

  it("aligns log navigation chrome with the system logs surface", () => {
    expect(source).not.toContain("IconRefresh")
    expect(source).not.toContain("PauseCircle")
    expect(source).not.toContain("hasOlder")
    expect(source).not.toContain("loadOlder")
    expect(source).not.toContain("pause,")
    expect(source).not.toContain("logs.historyExhausted")
    expect(source).not.toContain("logs.loadOlder")
    expect(source).toContain("showJumpToLatest")
    expect(source).toContain("onJumpToLatest={jumpToLatest}")
    expect(source).toContain('jumpToLatestLabel={t("logs.jumpToLatest")}')
    expect(source).not.toContain("TooltipTrigger")
    expect(source).not.toContain("ChevronDownIcon")
    expect(source).not.toContain('className="absolute bottom-4 right-4 radius-round"')
  })

  it("delegates terminal body, viewport, jump chrome, and footer shell to the shared live surface", () => {
    expect(source).toContain('from "@/components/shared/visualization/terminal-log-surface"')
    expect(source).toContain("<LiveLogSurface")
    expect(source).not.toContain("bg-[var(--terminal-log-background)] border-b")
    expect(source).not.toContain("font-mono h-full")
    expect(source).not.toContain('className="flex h-10 items-center justify-between border-t')
  })

  it("names manual scroll pause as scroll pause instead of log polling pause", () => {
    expect(source).toMatch(/if \(phase === "paused"\)\s+return t\("logs\.status\.scrollPaused"\)/)
  })

  it("uses the system-log footer shape for line count, source, status, and auto refresh", () => {
    expect(source).toContain("const DEFAULT_LINES = 100")
    expect(source).toContain("const LOG_WINDOW_OPTIONS = [100, 200, 500, 1000, 2000, 5000] as const")
    expect(source).toContain("windowSize")
    expect(source).toContain("lineWindow={{")
    expect(source).toContain('<span className="shrink-0 whitespace-nowrap">{lines.length} / {windowSize} {t("logs.toolbar.linesUnit")}</span>')
    expect(source).toContain('t("logs.toolbar.agentContainerSource", { container })')
    expect(source).toContain('className="flex min-w-0 basis-full items-center gap-2 sm:basis-auto sm:gap-4"')
    expect(source).toContain('className="flex shrink-0 gap-2 items-center whitespace-nowrap"')
    expect(source).toContain('<Switch')
    expect(source).toContain('checked={autoRefresh}')
    expect(source).toContain('onCheckedChange={setAutoRefresh}')
    expect(source).toContain('htmlFor="agent-log-auto-refresh"')
    expect(source).toContain('t("logs.toolbar.autoRefresh")')
  })
})
