import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-runtime-detail-drawer.tsx"), "utf8")
const utilsSource = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-runtime-detail-utils.ts"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-runtime-detail-layout.ts"), "utf8")
const overviewLayoutSource = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-overview-layout.ts"), "utf8")

describe("scan-runtime-detail-drawer contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ScanRuntimeDetailDrawer")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared status helpers and semantic surfaces", () => {
    expect(source).toContain("getScanStatusBadgeVariant")
    expect(source).toContain("getScanStatusClasses")
    expect(source).toContain("getScanStatusSurfaceClass")
    expect(source).toContain("getScanStatusTextClass")
    expect(source).not.toContain("function statusTone")
    expect(source).not.toContain("text-emerald-500")
    expect(source).not.toContain("text-amber-500")
    expect(source).not.toContain("bg-emerald-500")
    expect(source).not.toContain("bg-amber-500")
    expect(source).not.toContain("bg-black/5")
    expect(source).not.toContain("bg-background/50")
    expect(source).toContain('useTranslations("common.status")')
    expect(source).toContain("getRuntimeTaskPresentationStatus(task)")
    expect(source).toContain("statusLabel(status)")
    expect(source).toContain('t("runtimeDrawer.tasks.status.interrupted")')
    expect(source).toContain('t("runtimeDrawer.tasks.status.notStarted")')
    expect(source).not.toContain("{task.status}")
    expect(source).not.toContain("{status}\n          </Badge>")
  })

  it("uses shared tabs and the shared scan terminal log viewer for the structured detail workspace", () => {
    expect(source).toContain('from "@/components/shared/detail-drawer"')
    expect(source).toContain('from "@/components/scan/scan-log-list"')
    expect(source).toContain("DetailDrawer")
    expect(source).not.toContain("motion=")
    expect(source).toContain("DetailDrawerTabs")
    expect(source).toContain("DetailDrawerTabsList")
    expect(source).toContain("DetailDrawerTabsTrigger")
    expect(source).toContain("DetailDrawerTabsContent")
    expect(source).not.toContain("SheetContent")
    expect(source).not.toContain("<TabsList")
    expect(source).toContain("RuntimeLogsPanel")
    expect(source).toContain("<ScanLogList logs={logs} loading={logsLoading} />")
    expect(source).not.toContain("TableHeader")
    expect(source).not.toContain("TableRow")
    expect(source).not.toContain("TABLE_DENSE_CELL_RHYTHM_CLASS")
  })

  it("renders the target display name instead of the AIP resource name in the runtime header", () => {
    expect(source).toMatch(/scan\.target\?\.displayName \|\| scan\.target\?\.name \|\| t\("runtimeDrawer\.emptyTarget"\)/)
    expect(source).not.toContain('{scan.target?.name || t("runtimeDrawer.emptyTarget")}')
  })

  it("shows assignment provenance and deleted Agent state only in runtime detail", () => {
    expect(source).toContain('t("runtimeDrawer.meta.assignment")')
    expect(source).toContain('scan.assignmentMode === "pinned"')
    expect(source).toContain("scan.agentDeleted")
    expect(source).toContain("scan.agentName")
    expect(source).toContain("scan.agentStatus")
    expect(source).toContain("scan.agentHealthState")
    expect(source).toContain('t("runtimeDrawer.meta.stateUnavailable")')
    expect(source).not.toContain("useAgents")
    expect(source).not.toContain("agentsData")
    expect(source).not.toContain("assignedAgent")
  })

  it("renders runtime details with the same integrated tab panel used by scan overview", () => {
    expect(source).toContain("export function RuntimeConfigurationPanel")
    expect(source).toContain('from "@/components/shared/feedback/copy-button"')
    expect(source).toContain("serializeWorkflowConfiguration")
    expect(source).toContain('<section aria-label={t("runtimeDrawer.details.title")} className="min-w-0 overflow-hidden rounded-lg border border-border/60 bg-card">')
    expect(source).toContain("border-b border-border px-3")
    expect(source).toContain('<DetailDrawerTabsList variant="minimal" size="md" className="min-w-0 max-w-full overflow-x-auto">')
    expect(source).toContain('<DetailDrawerTabsTrigger variant="minimal" activeIndicator="fixed" value="logs" size="md">')
    expect(source).toContain("<RuntimeConfigurationPanel yamlContent={yamlContent} t={t} />")
    expect(source).toContain('<CopyButton')
    expect(source).toContain('copyLabel={t("runtimeDrawer.details.copyConfig")}')
    expect(source).toContain('toastId="scan-runtime-configuration-copy"')
    expect(source).toContain('"min-h-full min-w-0 p-4 pr-12 leading-6 text-foreground/90"')
    expect(source).not.toContain('<h3 className={textRole.sectionTitle}>{t("runtimeDrawer.details.title")}</h3>')
    expect(source).not.toContain('overflow-hidden rounded-lg border border-border/60">\n            <DetailDrawerTabsContent')
    expect(source).not.toContain("JSON.stringify(runtimeScan.configuration, null, 2)")
  })

  it("groups explicit progress logs into matching runtime task rows", () => {
    expect(source).toContain("buildTaskProgressLogsByTaskId")
    expect(source).toContain("taskProgressLogsByTaskId")
    expect(source).toContain("progressLogs: ScanLog[]")
    expect(source).toContain("runtimeScan.runtimeTasks")
    expect(utilsSource).toContain("log.taskId")
    expect(source).toContain("<RuntimeTaskProgressLines")
    expect(source).not.toContain("buildRuntimeTasks")
    expect(source).not.toContain("scan.stageProgress")
    expect(source).not.toContain("Object.entries(scan.stageProgress)")
  })

  it("renders expanded task progress through the shared copy-friendly raw viewer", () => {
    expect(source).toContain('from "@/components/shared/visualization/raw-log-viewer"')
    expect(source).toContain('from "@/components/scan/scan-log-list-state"')
    expect(source).toContain("<RawLogViewer")
    expect(source).toContain("content={buildScanLogContent(progressLogs)}")
    expect(source).toContain('className="h-auto max-h-48"')
    expect(source).not.toContain("getTerminalLogLevelBadgeStyle")
    expect(source).not.toContain("formatClockTime")
    expect(source).toContain('t("runtimeDrawer.tasks.noProgress")')
    expect(source).toContain('t("runtimeDrawer.tasks.failureTitle")')
  })

  it("uses localized engine descriptions in the collapsed task summary and keeps one expanded failure explanation", () => {
    expect(utilsSource).toContain("detail: localizedEngineDescriptionsByID.get(task.engineId)!")
    expect(utilsSource).toContain("failureSummary: getRuntimeTaskFailureSummary(task, scan, isCanonicalFailure)")
    expect(utilsSource).not.toContain("detail: task.error")
    expect(source).toContain("{task.detail}")
    expect(source).not.toContain('{task.detail || t("runtimeDrawer.tasks.noDetail")}')
    expect(utilsSource).toContain("failureDetail: task.failureDetail")
    expect(source).toContain("const failureMessage = [task.failureSummary, task.failureDetail].filter(Boolean).join(\" \")")
    expect(source).toContain("{failureMessage}")
    expect(source).not.toContain("logLines: buildRuntimeTaskFailureLines")
  })

  it("keeps runtime task row metadata in a stable right-side column", () => {
    const taskListSource = source.slice(source.indexOf("export function RuntimeTaskList"))

    expect(source).toContain('flex min-w-0 flex-col gap-3 px-4 py-3 lg:flex-row lg:items-center lg:justify-between')
    expect(source).toContain('min-w-0 flex-1 space-y-1.5')
    expect(source).toContain('max-w-2xl break-words", textRole.bodySubtle')
    expect(taskListSource).toContain('flex shrink-0 items-center gap-3 lg:justify-end')
    expect(taskListSource).toContain('space-y-1.5 lg:w-56')
    expect(taskListSource).toContain('grid grid-cols-[3.5rem_minmax(0,1fr)] items-baseline gap-3')
    expect(taskListSource).toContain('cn(metadataLabelClassName, "text-right")')
    expect(taskListSource).toContain('cn(textRole.metadataValue, "text-left tabular-nums whitespace-nowrap")')
    expect(taskListSource.indexOf('t("runtimeDrawer.tasks.duration")')).toBeLessThan(taskListSource.indexOf('t("runtimeDrawer.tasks.startedAt")'))
  })

  it("keeps runtime drawer content from widening the shared side panel", () => {
    expect(source).toContain('className="min-w-0 max-w-full space-y-6"')
    expect(source).toContain('className="min-w-0 flex-1 overflow-x-hidden overflow-y-auto px-6 py-5 [scrollbar-gutter:stable]"')
    expect(source).toContain('className="min-w-0 space-y-3"')
    expect(source).toContain('className="min-w-0 overflow-hidden rounded-lg border border-border/60 bg-card"')
    expect(source).toContain('from "@/components/scan/history/scan-runtime-detail-layout"')
    expect(source).toContain("RUNTIME_DETAIL_PANEL_CLASS")
    expect(source).toContain("RUNTIME_DETAIL_PANEL_FALLBACK_CLASS")
    expect(source).not.toContain("min-h-[260px]")
    expect(layoutSource).toContain('RUNTIME_DETAIL_PANEL_CLASS = "h-[320px] min-h-[260px]"')
    expect(layoutSource).toContain('RUNTIME_DETAIL_PANEL_FALLBACK_CLASS = "min-h-[260px]"')
    expect(source).toContain('"min-h-full min-w-0 p-4 pr-12 leading-6 text-foreground/90"')
  })

  it("renames the merged scan log tab to all-progress terminology", () => {
    expect(source).toContain('t("runtimeDrawer.details.allProgress")')
    expect(source).not.toContain('t("runtimeDrawer.details.logs")')
  })

  it("keeps runtime duration in the header metadata instead of the result preview strip", () => {
    expect(source).toContain("tasks: RuntimeTaskItem[]")
    expect(source).toContain("useRuntimeDurationNow(hasLiveRuntimeDuration(runtimeScan, runtimeTasks))")
    expect(source).toContain("<RuntimeHeader scan={runtimeScan} tasks={runtimeTasks} now={runtimeDurationNow} locale={locale} t={t} statusLabel={tStatus} />")
    expect(source).toContain("formatDurationClock(getRuntimeDurationSeconds(scan, tasks, now))")
    expect(source).not.toContain('key: "duration"')
    expect(source).not.toContain('t("runtimeDrawer.meta.progress")')
    expect(source).not.toContain('{scan.progress || 0}%')
  })

  it("uses drawer open state as the runtime detail refresh boundary", () => {
    expect(source).toContain("refreshEnabled: open")
  })

  it("renders running task duration from the live runtime clock", () => {
    expect(source).toContain("export function hasLiveRuntimeDuration")
    expect(source).toContain("export function useRuntimeDurationNow")
    expect(source).toContain("RUNTIME_DURATION_REFRESH_MS")
    expect(source).toContain("window.setInterval")
    expect(source).toContain("const duration = getRuntimeTaskDurationSeconds(task, now)")
    expect(source).toContain("formatDurationClock(duration)")
    expect(source).not.toContain("formatDurationClock(task.duration)")
  })

  it("keeps the full local calendar date in task start timestamps", () => {
    expect(source).toContain('formatLocalTimestampSeconds(value)')
    expect(source).toContain('{formatTaskStartTime(task.startedAt)}')
    expect(source).not.toContain("toLocaleTimeString")
  })

  it("keeps duplicate risk insight out of the runtime header", () => {
    expect(source).not.toContain("runtimeDrawer.riskInsight")
    expect(source).toContain('t("runtimeDrawer.summary.risks")')
  })

  it("shows the bound workflow as runtime header metadata", () => {
    expect(source).toContain('t("runtimeDrawer.meta.workflow")')
    expect(source).toContain("useScanWorkflows")
    expect(source).toContain("getScanWorkflowDisplayName")
    expect(source).toContain("function RuntimeTaskList")
    expect(source).toContain("task.title")
  })

  it("turns the runtime summary strip into an adaptive result preview", () => {
    expect(source).not.toContain('t("runtimeDrawer.summary.assetScale")')
    expect(source).not.toContain("assetScale")
    expect(source).not.toContain("2xl:grid-cols-5")
    expect(source).not.toContain("xl:grid-cols-5")
    expect(source).toContain("overflow-hidden rounded-lg border border-border/60 bg-card")
    expect(source).toContain("SCAN_OVERVIEW_SUMMARY_GRID_CLASS")
    expect(overviewLayoutSource).toContain("grid min-w-0 grid-cols-2 sm:grid-cols-6")
    expect(source).not.toContain('index === summaryItems.length - 1 && "col-span-2 sm:col-span-1"')
    expect(source).toContain("SCAN_OVERVIEW_SUMMARY_ITEM_CLASS")
    expect(overviewLayoutSource).toContain("flex min-w-0 items-baseline justify-between gap-1.5 px-3 py-2 sm:justify-center")
    expect(source).toContain('t("runtimeDrawer.summary.tasks")')
    expect(source).toContain('t("runtimeDrawer.summary.taskCount"')
    expect(source).toContain('t("cards.subdomains")')
    expect(source).toContain('t("cards.websites")')
    expect(source).toContain('t("cards.ips")')
    expect(source).toContain('t("cards.urls")')
    expect(source).toContain('t("runtimeDrawer.summary.risks")')
  })

  it("keeps zero-failure counts out of the runtime summary strip", () => {
    expect(source).not.toContain("failedCount")
    expect(source).not.toContain('t("runtimeDrawer.summary.failed")')
    expect(source).not.toContain('key: "failed"')
  })

  it("places the asset result summary inside the runtime side panel", () => {
    expect(source).toContain("function RuntimeAssetSummaryPanel")
    expect(source).toContain("<RuntimeAssetSummaryPanel scan={scan} t={t} />")
    expect(source).toContain('t("runtimeDrawer.assets.title")')
    expect(source).not.toContain("RuntimeAssetSummaryStrip")
    expect(source).not.toContain('t("runtimeDrawer.assets.viewResults")')
  })

  it("keeps agent and duration metadata in the runtime header instead of the side panel", () => {
    expect(source).toContain('t("runtimeDrawer.meta.agent")')
    expect(source).toContain('t("runtimeDrawer.summary.duration")')
    expect(source).not.toContain('<RuntimeSidePanelRow label={t("runtimeDrawer.meta.agent")')
    expect(source).not.toContain('<RuntimeSidePanelRow\n            label={t("runtimeDrawer.summary.duration")')
  })

  it("keeps scan configuration out of the runtime side panel", () => {
    expect(source).not.toContain("getConfigurationValue")
    expect(source).not.toContain('t("runtimeDrawer.sidePanel.config")')
    expect(source).not.toContain('t("runtimeDrawer.sidePanel.concurrency")')
    expect(source).not.toContain('t("runtimeDrawer.sidePanel.timeout")')
    expect(source).not.toContain('t("runtimeDrawer.sidePanel.retries")')
    expect(source).not.toContain('t("runtimeDrawer.sidePanel.viewFullConfig")')
  })

  it("renders runtime logs without the inline search and level filter toolbar", () => {
    expect(source).not.toContain('from "@/components/ui/input"')
    expect(source).not.toContain('from "@/components/ui/select"')
    expect(source).not.toContain('t("runtimeDrawer.details.searchLogs")')
    expect(source).not.toContain('t("runtimeDrawer.details.allLevels")')
    expect(source).not.toContain("filteredLogs")
    expect(source).not.toContain("logs.map((log)")
  })

  it("keeps copy-log actions out of the runtime detail tab header", () => {
    expect(source).not.toContain("function handleCopyLogs")
    expect(source).not.toContain("RuntimeOverviewSidePanel({\n  scan,\n  locale,\n  t,\n  logText,\n  onCopyLogs,")
    expect(source).not.toContain("scan-runtime-log-copy")
    expect(source).not.toContain('disabled={!logText}')
    expect(source).not.toContain('<Copy className="h-3.5 w-3.5" />')
  })

  it("keeps quick actions out of the runtime overview side panel", () => {
    expect(source).not.toContain("runtimeDrawer.sidePanel.quickActions")
    expect(source).not.toContain("runtimeDrawer.quickActions")
    expect(source).not.toContain("const quickActions")
  })

  it("pins the runtime overview side panel at the xl desktop breakpoint", () => {
    expect(source).toContain("SCAN_OVERVIEW_STICKY_SIDE_PANEL_CLASS")
    expect(overviewLayoutSource).toContain('`${SCAN_OVERVIEW_SIDE_PANEL_CLASS} xl:sticky xl:top-4`')
    expect(source).not.toContain("2xl:sticky")
    expect(source).not.toContain("2xl:w-80")
  })

  it("keeps asset results out of the runtime detail tabs", () => {
    expect(source).not.toContain('value="assets"')
    expect(source).not.toContain('t("runtimeDrawer.details.assets")')
    expect(source).not.toContain("RuntimeAssetsPanel")
  })

  it("keeps debug information out of the runtime detail tabs", () => {
    expect(source).not.toContain('value="debug"')
    expect(source).not.toContain('t("runtimeDrawer.details.debug")')
  })

  it("uses status icons instead of numeric badges for runtime task timeline nodes", () => {
    expect(source).toContain("getScanStatusSurfaceClass(task.status)")
    expect(source).toContain('presentationStatus === "not_started"')
    expect(source).toContain('"border border-border bg-card"')
    expect(source).toContain("{statusIcon(presentationStatus)}")
    expect(source).toContain('className="absolute bottom-[-16px] left-3 top-10 w-px bg-border"')
    expect(source).not.toContain('className="absolute bottom-[-16px] left-3 top-8 w-px bg-border"')
    expect(source).not.toContain("{index + 1}")
    expect(source).not.toContain("border-success/30 bg-background text-xs font-medium tabular-nums text-success")
  })

  it("uses canonical status glyphs for runtime success, failure, and cancellation", () => {
    expect(source).toContain("semanticIcons.status.success")
    expect(source).toContain("semanticIcons.status.failed")
    expect(source).toContain("semanticIcons.status.cancelled")
    expect(source).not.toContain("return <IconCircleX")
    expect(source).not.toContain("return <IconCircleCheck")
  })

  it("uses a structured handoff skeleton instead of raw rectangle loading blocks", () => {
    expect(source).toContain("ContentHandoff")
    expect(source).toContain("ScanRuntimeDetailDrawerLoadingState")
    expect(source).toContain("const isInitialLoading = (isLoading && !runtimeScan) || (Boolean(runtimeScan) && executedEngineDisplay.isLoading)")
    expect(source).toContain("const showErrorState = !isInitialLoading && (Boolean(error) || Boolean(executedEngineDisplay.error) || !runtimeScan)")
    expect(source).toContain('owner="scan-runtime-detail-drawer-content"')
    expect(source).toContain("getLoadingOwnerAttributes")
    expect(source).toContain('getLoadingOwnerAttributes({ owner: "scan-runtime-detail-drawer", layer: "interaction", intent: "interaction" })')
    expect(source).toContain('<DetailDrawerTabs value="logs" aria-hidden="true">')
    expect(source).toContain('<DetailDrawerTabsList variant="minimal" size="md" className="min-w-0 max-w-full overflow-x-auto">')
    expect(source).toContain('<DetailDrawerTabsTrigger variant="minimal" activeIndicator="fixed" value="logs" size="md" disabled>')
    expect(source).toContain('<DetailDrawerTabsTrigger variant="minimal" activeIndicator="fixed" value="config" size="md" disabled>')
    expect(source).not.toContain('<Skeleton className="h-56 w-full" />')
    expect(source).not.toContain('<Skeleton className="h-64 w-full" />')
    expect(source).not.toContain('<Skeleton className="h-8 w-20 rounded-md" />')
    expect(source).not.toContain('<Skeleton className="h-9 w-24 rounded-md" />')
  })
})
