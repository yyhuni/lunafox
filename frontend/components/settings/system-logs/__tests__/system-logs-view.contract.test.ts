import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/system-logs/system-logs-view.tsx"), "utf8")
const loadingSource = readFileSync(path.resolve(process.cwd(), "components/settings/system-logs/system-logs-loading-state.tsx"), "utf8")
const layoutSource = readFileSync(
  path.resolve(process.cwd(), "components/settings/system-logs/system-logs-layout.ts"),
  "utf8"
)

describe("system-logs-view contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function SystemLogsView")
    expect(loadingSource).toContain("export function SystemLogsLoadingState")
    expect(source).toContain("pageTitle: string")
    expect(source).toContain("pageDescription: string")
    expect(source).toContain("title={pageTitle}")
    expect(source).toContain("description={pageDescription}")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
    expect(source).not.toContain("SystemLogsPageSkeleton")
    expect(source).not.toContain("system-logs-page-skeleton")
  })

  it("keeps the log source fixed while removing file switching from the toolbar", () => {
    expect(source).toContain("toolbar.serverSource")
    expect(source).not.toContain("lunafox.log")
    expect(source).not.toContain("selectedFile")
    expect(source).not.toContain("onFileChange=")
  })

  it("uses the shared structured log viewer controls", () => {
    expect(source).toContain("const [windowSize, setWindowSize] = useState<number>(SYSTEM_LOGS_DEFAULT_LINES)")
    expect(source).toContain("const LOG_WINDOW_OPTIONS = [100, 200, 500, 1000, 2000, 5000] as const")
    expect(layoutSource).toContain("export const SYSTEM_LOGS_DEFAULT_LINES = 100")
    expect(loadingSource).toContain("options: [100, 200, 500, 1000, 2000, 5000]")
    expect(source).toContain("StructuredLogViewer")
    expect(source).toContain("<StructuredLogViewer viewportRef={viewportRef}")
    expect(source).toContain("filterStructuredLogLines")
    expect(source).toContain("formatStructuredLogLine")
    expect(source).toContain('from "@/components/shared/visualization/terminal-log-copy-all-button"')
    expect(source).toContain("const filteredLogText")
    expect(source).toContain("topRightAction={<TerminalLogCopyAllButton")
    expect(source).toContain("value={filteredLogText}")
    expect(source).toContain('copyLabel={t("logs.copyVisible")}')
    expect(source).toContain('toastId="system-log-copy-all"')
    expect(source).toContain("TerminalLogToolbar")
    expect(source).toContain("lineWindow={{")
    expect(source).not.toContain("function LogToolbar")
    expect(source).not.toContain("handleDownload")
    expect(source).not.toContain("setLines")
    expect(source).not.toContain("setLogLevel")
    expect(source).not.toContain("toolbar.download")
  })

  it("keeps history loading out of the system log chrome", () => {
    expect(source).toContain('<span className="shrink-0 whitespace-nowrap">{logLines.length} / {windowSize} {t("toolbar.linesUnit")}</span>')
    expect(layoutSource).toContain('"flex min-w-0 basis-full items-center gap-2 sm:basis-auto sm:gap-4"')
    expect(layoutSource).toContain('"flex shrink-0 items-center gap-2 whitespace-nowrap"')
    expect(source).toMatch(/if \(phase === "paused"\)\s+return t\("logs\.status\.scrollPaused"\)/)
    expect(source).not.toContain("logs.historyExhausted")
    expect(source).not.toContain("logs.loadOlder")
    expect(source).not.toContain("onClick={() => void loadOlder()}")
    expect(source).not.toContain("PauseCircle")
    expect(source).not.toContain("onClick={pause}")
    expect(source).not.toContain("aria-label={t(\"logs.status.scrollPaused\")}")
  })

  it("renders jump-to-latest as a contextual floating affordance", () => {
    expect(source).toContain("const showJumpToLatest = !isInitialLoading && (!autoScroll || hasNewer)")
    expect(source).toContain("showJumpToLatest={showJumpToLatest}")
    expect(source).toContain("onJumpToLatest={jumpToLatest}")
    expect(source).toContain('jumpToLatestLabel={t("logs.jumpToLatest")}')
    expect(source).not.toContain("TooltipTrigger")
    expect(source).not.toContain("ChevronDownIcon")
  })

  it("lets the log viewport fill the remaining page space", () => {
    expect(layoutSource).toContain("flex min-h-0 flex-1 flex-col px-4 lg:px-6")
    expect(layoutSource).toContain("flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border")
    expect(source).toContain("SYSTEM_LOGS_CONTENT_SHELL_CLASS")
    expect(source).toContain("SYSTEM_LOGS_TERMINAL_SHELL_CLASS")
    expect(source).toContain("<LiveLogSurface")
    expect(source).not.toContain("bg-[#1e1e1e]")
    expect(source).not.toContain("min-h-[400px]")
  })

  it("keeps long mock log payloads inside the terminal viewport", () => {
    expect(layoutSource).toContain("flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border")
    expect(source).toContain("SYSTEM_LOGS_TERMINAL_SHELL_CLASS")
    expect(source).toContain("viewportRef={viewportRef}")
    expect(source).toContain("StructuredLogViewer")
  })

  it("derives resolved and loading log chrome from the same layout contract", () => {
    expect(layoutSource).toContain("SYSTEM_LOGS_PAGE_SHELL_CLASS")
    expect(layoutSource).toContain("SYSTEM_LOGS_HEADER_OVERLAY_SHELL_CLASS")
    expect(layoutSource).toContain("SYSTEM_LOGS_CONTENT_SHELL_CLASS")
    expect(layoutSource).toContain("SYSTEM_LOGS_TERMINAL_SHELL_CLASS")
    expect(layoutSource).not.toContain("SYSTEM_LOGS_TERMINAL_BODY_CLASS")
    expect(layoutSource).not.toContain("SYSTEM_LOGS_TERMINAL_VIEWPORT_CLASS")
    expect(layoutSource).not.toContain("SYSTEM_LOGS_FOOTER_CLASS")
    expect(layoutSource).toContain("SYSTEM_LOGS_FOOTER_STATUS_GROUP_CLASS")
    expect(layoutSource).toContain("SYSTEM_LOGS_FOOTER_REFRESH_GROUP_CLASS")
    expect(layoutSource).toContain("SYSTEM_LOGS_FOOTER_OVERLAY_STATUS_GROUP_CLASS")
    expect(layoutSource).toContain("SYSTEM_LOGS_FOOTER_OVERLAY_REFRESH_GROUP_CLASS")

    expect(source).toContain("from \"@/components/settings/system-logs/system-logs-layout\"")
    expect(source).toContain("SYSTEM_LOGS_PAGE_SHELL_CLASS")
    expect(source).toContain("SYSTEM_LOGS_TERMINAL_SHELL_CLASS")
    expect(source).toContain('from "@/components/shared/visualization/terminal-log-toolbar"')
    expect(source).toContain('from "@/components/shared/visualization/terminal-log-surface"')
    expect(source).toContain("<TerminalLogToolbar")
    expect(source).toContain("<LiveLogSurface")
    expect(loadingSource).toContain("<LiveLogSurface")
    expect(loadingSource).toContain("SYSTEM_LOGS_FOOTER_OVERLAY_STATUS_GROUP_CLASS")
    expect(loadingSource).toContain("SYSTEM_LOGS_FOOTER_OVERLAY_REFRESH_GROUP_CLASS")
    expect(source).not.toContain('className="flex flex-1 items-center gap-3"')
    expect(source).not.toContain('className="flex items-center gap-2"')
  })

  it("pairs the route-critical terminal regions across loading and content", () => {
    for (const slot of [
      "system-logs-header",
      "system-logs-terminal-toolbar",
      "system-logs-log-surface",
    ]) {
      expect(loadingSource).toContain(`getLoadingStructureSlotAttributes(\"${slot}\")`)
      expect(source).toContain(`getLoadingStructureSlotAttributes(\"${slot}\")`)
    }
  })

  it("keeps toolbar loading controls derived from resolved control shells", () => {
    expect(loadingSource).toContain("<TerminalLogToolbar")
    expect(loadingSource).toContain('searchTerm=""')
    expect(loadingSource).toContain('levelFilter="all"')
    expect(loadingSource).toContain("loading")

    expect(loadingSource).not.toContain("Skeleton className=\"h-8 w-full rounded-sm\"")
    expect(loadingSource).not.toContain("Skeleton key={index} className=\"h-8 w-12 rounded-sm\"")
  })

  it("keeps footer auto-refresh loading control on the real switch primitive", () => {
    expect(source).toContain("<Switch id=\"auto-refresh\"")
    expect(source).toContain("import { Switch } from \"@/components/ui/switch\"")
    expect(loadingSource).toContain("SYSTEM_LOGS_FOOTER_OVERLAY_REFRESH_GROUP_CLASS")
    expect(loadingSource).toContain('<Switch checked disabled tabIndex={-1} className="scale-75" />')
    expect(loadingSource).not.toContain('Skeleton className="h-4 w-7 rounded-full"')
  })

  it("keeps the component-owned loading state independent from live log data", () => {
    expect(loadingSource).not.toContain("use-system-logs")
    expect(loadingSource).not.toContain("structured-log-viewer")
  })

  it("waits for the first log request to leave idle before releasing the route handoff", () => {
    expect(source).toContain('if (phase === "idle" || isInitialLoading)')
    expect(source).toContain('}, [isInitialLoading, onReady, phase]);')
  })
})
