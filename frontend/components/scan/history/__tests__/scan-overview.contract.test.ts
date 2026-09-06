import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-overview.tsx"), "utf8")
const guide = readFileSync(path.resolve(process.cwd(), "components/scan/history/README.md"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-overview-layout.ts"), "utf8")

describe("scan-overview contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ScanOverview")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared content handoff for scan overview loading", () => {
    expect(source).toContain("ContentHandoff")
    expect(source).toContain('owner="scan-overview-content"')
    expect(source).toContain("skeleton={<ScanOverviewLoadingState />}")
    expect(source).toContain("useDetailShellReadySignal")
    expect(source).toContain("detailShellReady?.deferInitialSkeleton")
    expect(source).toContain("if (detailShellReady?.deferInitialSkeleton && isInitialLoading) {")
  })

  it("renders the overview as a page-level runtime detail workspace", () => {
    expect(source).toContain("buildTaskProgressLogsByTaskId")
    expect(source).toContain("scan.runtimeTasks")
    expect(source).not.toContain("buildRuntimeTasks")
    expect(source).toContain("taskProgressLogsByTaskId")
    expect(source).toContain("RuntimeHeader")
    expect(source).toContain("RuntimeSummarySection")
    expect(source).toContain("RuntimeTaskList")
    expect(source).toContain("ScanOverviewRuntimeDetails")
    expect(source).toContain("RuntimeOverviewSidePanel")
    expect(source).not.toContain("buildScanLogContent")
    expect(source).not.toContain("copyTextToClipboard")
    expect(source).not.toContain("scan-overview-log-copy")
    expect(source).not.toContain("RuntimeAssetSummaryStrip")
    expect(source).not.toContain("<ScanOverviewAssets")
    expect(source).not.toContain("<ScanStageProgress")
    expect(source).not.toContain("<ScanVulnerabilitySummary")
    expect(source).not.toContain("<ScanLogsPanel")
  })

  it("uses the shared runtime task expansion for terminal diagnostics", () => {
    const drawer = readFileSync(
      path.resolve(process.cwd(), "components/scan/history/scan-runtime-detail-drawer.tsx"),
      "utf8"
    )

    expect(source).toContain("<RuntimeTaskList")
    expect(drawer).toContain("function RuntimeTaskDiagnostics")
    expect(drawer).toContain("<RuntimeTaskDiagnostics diagnostics={task.diagnostics} t={t} />")
    expect(drawer).not.toContain("DiagnosticDrawer")
  })

  it("uses all-progress terminology for the merged runtime log view", () => {
    const drawer = readFileSync(
      path.resolve(process.cwd(), "components/scan/history/scan-runtime-detail-drawer.tsx"),
      "utf8"
    )

    expect(drawer).toContain('t("runtimeDrawer.details.allProgress")')
  })

  it("uses the desktop workbench side panel from the xl breakpoint", () => {
    expect(source).toContain("SCAN_OVERVIEW_WORKBENCH_CLASS")
    expect(layoutSource).toContain("flex min-h-0 min-w-0 flex-1 flex-col gap-4 xl:flex-row xl:items-start")
    expect(source).not.toContain("2xl:flex-row")
  })

  it("documents the scan overview detail-shell handoff contract", () => {
    expect(guide).toContain("## Detail Overview Shell Handoff")
    expect(guide).toContain("ScanHistoryDetailShellLoadingState")
    expect(guide).toContain("scan-history-detail-shell-layout.tsx")
    expect(guide).toContain("useDetailShellReadySignal")
    expect(guide).toContain("deferInitialSkeleton")
  })

  it("documents the runtime overview side panel breakpoint contract", () => {
    expect(guide).toContain("## Runtime Overview Side Panel")
    expect(guide).toContain("Tailwind `xl` (`1280px`)")
    expect(guide).toContain("fixed auxiliary width of `w-80`")
    expect(guide).toContain("this breakpoint back to `2xl`")
  })
})
