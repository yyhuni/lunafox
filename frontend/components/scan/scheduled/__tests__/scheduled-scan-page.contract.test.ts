import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/scheduled-scan-page.tsx"), "utf8")
const dataTableSource = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/scheduled-scan-data-table.tsx"), "utf8")
const sectionsSource = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/scheduled-scan-page-sections.tsx"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/scheduled-scan-page-layout.ts"), "utf8")

describe("scheduled-scan-page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default function ScheduledScanPage")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("keeps the scheduled scan page aligned to the approved scheduling workbench layout", () => {
    expect(source).toContain("export interface ScheduledScanPageProps")
    expect(source).toContain("onReady?: () => void")
    expect(source).toContain("deferInitialSkeleton?: boolean")
    expect(source).not.toContain("ScheduledScanSummaryStrip")
    expect(source).toContain("ScheduledScanTimeline")
    expect(source).not.toContain("ScheduledScanFrequencyDistribution")
    expect(source).toContain("ScheduledScanPageLoadingState")
    expect(sectionsSource).not.toContain("ScheduledScanSummaryStripShell")
    expect(sectionsSource).toContain("ScheduledScanOverviewSectionShell")
    expect(sectionsSource).toContain("ScheduledScanTableSectionShell")
    expect(sectionsSource).toContain("ScheduledScanPageHeader")
    expect(sectionsSource).toContain("ScheduledScanInsightGrid")
    expect(sectionsSource).toContain("export function ScheduledScanPageLoadingState")
    expect(source).toContain("from \"@/components/scan/scheduled/scheduled-scan-data-table\"")
    expect(source).toContain("onAddNew={handleAddNew}")
    expect(source).not.toContain("scheduled-scan-page-skeleton")
  })

	it("uses the independent server overview and retains explicit overview failure states", () => {
		expect(source).toContain("useScheduledScanOverviewSummary")
		expect(source).toContain("ScheduledScanOverviewLoadingState")
		expect(source).toContain("ScheduledScanOverviewFailure")
		expect(source).toContain("ScheduledScanOverviewStaleNotice")
		expect(source).toContain("overview.isInitialError")
		expect(source).toContain("overview.isRefetchStale")
		expect(source).toContain("overview.data.next24HoursScheduledScanCount")
		expect(source).not.toContain("pageSize: 1000")
		expect(source).not.toContain("buildScheduledScanInsights")
		expect(sectionsSource).toContain("ScheduledScanOverviewUpcoming")
		expect(sectionsSource).toContain("next24HoursCount")
		expect(sectionsSource).toContain("scan.displayName")
	})

  it("derives scheduled scan loading geometry from the resolved page layout owner", () => {
    expect(source).toContain("SCHEDULED_SCAN_PAGE_SIZE")
    expect(source).toContain("React.useState(SCHEDULED_SCAN_PAGE_SIZE)")
    expect(source).toContain("pageSize || SCHEDULED_SCAN_PAGE_SIZE")
    expect(dataTableSource).toContain("SCHEDULED_SCAN_PAGE_SIZE")
    expect(dataTableSource).toContain("pageSize = SCHEDULED_SCAN_PAGE_SIZE")
    expect(sectionsSource).toContain("SCHEDULED_SCAN_TIMELINE_BODY_CLASS")
    expect(sectionsSource).not.toContain("SCHEDULED_SCAN_FREQUENCY_BODY_CLASS")
    expect(sectionsSource).toContain("SCHEDULED_SCAN_TABLE_LOADING_ROW_HEIGHT_PX")
    expect(sectionsSource).toContain("getDataTableSkeletonRowCount(SCHEDULED_SCAN_PAGE_SIZE)")
  })

  it("keeps scheduled scan loading rows on the actual dense table rhythm", () => {
    expect(layoutSource).toContain("TABLE_DENSE_ROW_RHYTHM_HEIGHT_PX")
    expect(layoutSource).toContain(
      "SCHEDULED_SCAN_TABLE_LOADING_ROW_HEIGHT_PX = TABLE_DENSE_ROW_RHYTHM_HEIGHT_PX"
    )
    expect(layoutSource).not.toContain("TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX")
  })

  it("passes all/enabled/paused counts into the quick filter tabs", () => {
    expect(source).toContain("const quickFilterCounts = React.useMemo")
    expect(source).toContain("counts.all += 1")
    expect(source).toContain("counts.enabled += 1")
    expect(source).toContain("counts.paused += 1")
    expect(source).toContain("quickFilterCounts={quickFilterCounts}")
  })

  it("passes selected scheduled scans into the table bulk delete flow", () => {
    expect(source).toContain("useBatchDeleteScheduledScans")
    expect(source).toContain("const [selectedScheduledScans, setSelectedScheduledScans]")
    expect(source).toContain("const handleBulkDelete = React.useCallback")
    expect(source).toContain("await batchDeleteScheduledScans(ids)")
    expect(source).toContain("setSelectedScheduledScans([])")
    expect(source).toContain("onBulkDelete={handleBulkDelete}")
    expect(source).toContain("onSelectionChange={setSelectedScheduledScans}")
    expect(source).toContain("selectedRows={selectedScheduledScans}")
  })

  it("exposes explicit global batch status actions with disable confirmation and recovery", () => {
    expect(source).toContain("useBatchUpdateScheduledScanStatus")
    expect(source).toContain("selectedRowActions={selectedRowActions}")
    expect(source).toContain('tScan("scheduled.batchStatus.enable")')
    expect(source).toContain('tScan("scheduled.batchStatus.disable")')
    expect(source).toContain('variant="destructive"')
    expect(source).toContain('tScan("scheduled.batchStatus.disableDescription"')
    expect(source).toContain("setSelectedScheduledScans([])")
    expect(dataTableSource).toContain("selectedRowActions?: SelectedRowActionBarAction[]")
  })

  it("uses the scheduled scan API cursor contract for the management list", () => {
    expect(source).toContain("const [pageTokens, setPageTokens]")
    expect(source).toContain("pageToken,")
    expect(source).toContain("getCursorPaginationNavigation")
    expect(source).toContain("getCursorPageTransition")
    expect(source).toContain("cursorPaginationSummary={{ total: displayedTotal }}")
    expect(source).toContain("paginationNavigation={paginationNavigation}")
    expect(source).not.toContain("Math.ceil(total")
    expect(source).not.toContain("totalPages={displayedTotalPages}")
  })

  it("routes scheduled scan row activation through the existing edit callback", () => {
    expect(source).toContain("onRowClick={handleEdit}")
    expect(dataTableSource).toContain("onRowClick?: (scan: ScheduledScan) => void")
    expect(dataTableSource).toContain("onRowClick: onRowClick")
    expect(dataTableSource).toContain("(row) => onRowClick(row as ScheduledScan)")
  })

  it("reuses the scheduled scan page loading state through the shared handoff during initial loading", () => {
    expect(source).toContain("ContentHandoff")
    expect(source).toContain("isLoading={isLoading}")
    expect(source).toContain('owner="scheduled-scan-page-content"')
    expect(source).toContain("skeleton={<ScheduledScanPageLoadingState stableSurfaceRowCount={stableSurfaceRowCount} />}")
    expect(source).not.toContain("owner=\"scheduled-scan-page-initial\"")
  })

  it("keeps scheduled scan loading state inside the resolved section owner", () => {
    expect(sectionsSource).toContain("export function ScheduledScanPageLoadingState")
    expect(sectionsSource).toContain("export function ScheduledScanDataTableLoadingState")
    expect(sectionsSource).not.toContain("ScheduledScanSummaryStripLoadingState")
    expect(sectionsSource).toContain("ScheduledScanTimelineLoadingState")
    expect(sectionsSource).not.toContain("ScheduledScanFrequencyDistributionLoadingState")
    expect(sectionsSource).toContain("ScheduledScanDataTable")
    expect(sectionsSource).toContain("createScheduledScanColumns")
    expect(sectionsSource).toContain("loading")
    expect(sectionsSource).toContain("initialLoading")
    expect(sectionsSource).toContain("loadingRowCount={rows}")
    expect(sectionsSource).toContain("loadingRowHeightEstimate={SCHEDULED_SCAN_TABLE_LOADING_ROW_HEIGHT_PX}")
    expect(sectionsSource).toContain("pageSize={SCHEDULED_SCAN_PAGE_SIZE}")
    expect(sectionsSource).toContain('paginationNavigation={{ mode: "cursor", canFirstPage: false, canPreviousPage: false, canNextPage: false }}')
    expect(sectionsSource).not.toContain("ScheduledScanSummaryStripShell")
    expect(sectionsSource).toContain("ScheduledScanOverviewSectionShell")
    expect(sectionsSource).toContain('getLoadingStructureSlotAttributes("scheduled-scan-table")')
    for (const slot of [
      "scheduled-scan-header",
      "scheduled-scan-insight-grid",
      "scheduled-scan-table",
    ]) {
      expect(sectionsSource).toContain(`getLoadingStructureSlotAttributes("${slot}")`)
    }
    expect(source).toContain("ScheduledScanTableSectionShell")
    expect(source).toContain("ScheduledScanPageHeader")
    expect(source).toContain("ScheduledScanInsightGrid")
    expect(sectionsSource).not.toContain("ScheduledScanPageSkeleton")
    expect(sectionsSource).not.toContain("ScheduledScanDataTableSkeleton")
    expect(sectionsSource).not.toContain("onAddNew={() => {}}")
    expect(sectionsSource).not.toContain("onSearch={() => {}}")
    expect(sectionsSource).not.toContain("onPageChange={() => {}}")
    expect(sectionsSource).not.toContain("onPageSizeChange={() => {}}")
  })

  it("keeps localized column definitions available to the initial table geometry", () => {
    expect(sectionsSource).toContain('taskName: tColumns("scheduledScan.taskName")')
    expect(sectionsSource).toContain('scope: tScan("scheduled.workbench.targetColumn")')
    expect(sectionsSource).toContain('lastRun: tColumns("scheduledScan.lastRun")')
    expect(sectionsSource).not.toContain('taskName: ""')
    expect(sectionsSource).not.toContain('scope: ""')
  })

  it("reports readiness only after the first scheduled scan data frame is available", () => {
    expect(source).toContain("export default function ScheduledScanPage({")
    expect(source).toContain("onReady,")
    expect(source).toContain("deferInitialSkeleton = false,")
    expect(source).toContain("}: ScheduledScanPageProps)")
    expect(source).toContain("const hasPrimaryDataFrame = !isLoading")
    expect(source).toContain("if (!hasPrimaryDataFrame) return")
    expect(source).toContain("onReady?.()")
    expect(source).toContain("const [isRouteBoundaryEntry] = React.useState(deferInitialSkeleton)")
  })

  it("does not introduce a second visible table-level loading owner after the page skeleton exits", () => {
    expect(source).toContain("if (isLoading && deferInitialSkeleton) return null")
    expect(source).toContain("if (isRouteBoundaryEntry)")
    expect(source).toContain("return pageContent")
    expect(source).not.toContain("return <ScheduledScanPageLoadingState />")
    expect(source).not.toContain("owner=\"scheduled-scan-page-table\"")
    expect(source).not.toContain("loading: () => <ScheduledScanDataTableLoadingState")
  })

  it("does not show a dialog-shaped loading state before opening the create scheduled scan sheet", () => {
    expect(source).toContain("loadCreateScheduledScanSheet")
    expect(source).toContain("CreateScheduledScanSheet")
    expect(source).toContain("EditScheduledScanDialog = dynamic")
    expect(source).toContain("deferredInteractionUnmountDelayMs")
    expect(source).toContain("shouldMountCreateDialog")
    expect(source).toContain("shouldMountEditDialog")
    expect(source).not.toContain('owner="scheduled-scan-create-dialog-loading"')
    expect(source).not.toContain('owner="scheduled-scan-edit-dialog-loading"')
    expect(source).not.toContain('openAfterLoad("edit-dialog"')
  })
})
