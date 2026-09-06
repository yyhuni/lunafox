import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/organization-detail-view-sections.tsx"), "utf8")
const enMessages = JSON.parse(readFileSync(path.resolve(process.cwd(), "messages/en.json"), "utf8"))
const zhMessages = JSON.parse(readFileSync(path.resolve(process.cwd(), "messages/zh.json"), "utf8"))

function readNestedMessage(messages: Record<string, unknown>, key: string) {
  return key.split(".").reduce<unknown>((current, segment) => {
    if (!current || typeof current !== "object") return undefined
    return (current as Record<string, unknown>)[segment]
  }, messages)
}

describe("organization-detail-view-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function OrganizationDetailViewLoadingState")
    expect(source).toContain("export function OrganizationDetailViewWorkbench")
    expect(source).toContain("ScheduledScanDataTable")
    expect(source).toContain("TabsList variant=\"content\"")
    expect(source).toContain("presetOrganizationId")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses semantic icons for organization detail actions and metrics without owning a local error state", () => {
    expect(source).toContain('import { semanticIcons } from "@/components/icons"')
    expect(source).toContain("semanticIcons.action.edit")
    expect(source).toContain("semanticIcons.action.refresh")
    expect(source).toContain("semanticIcons.concept.target")
    expect(source).toContain("semanticIcons.concept.scheduledScan")
    expect(source).not.toContain("AppErrorState")
    expect(source).not.toContain("error.message")
  })

  it("keeps add-target actions scoped to the targets table toolbar", () => {
    expect(source).not.toContain("const AddIcon = semanticIcons.action.add")
    expect(source).not.toContain("<AddIcon")
    expect(source).toContain("onAddNew={state.handleAddTarget}")
    expect(source).toContain('addButtonText={state.tTarget("addTarget")}')
  })

  it("uses a split workbench panel for organization edit and target linking", () => {
    expect(source).toContain("activeActionPanel")
    expect(source).toContain("EditOrganizationFormPanel")
    expect(source).toContain("LinkTargetFormPanel")
    expect(source).toContain("lg:flex-row")
    expect(source).toContain("lg:max-w-lg")
    expect(source).not.toContain("<EditOrganizationDialog")
    expect(source).not.toContain("<AddTargetDialog")
  })

  it("keeps scheduled scans data loading on the resolved table owner", () => {
    expect(source).toContain('from "@/components/scan/scheduled/scheduled-scan-data-table"')
    expect(source).not.toContain("const ScheduledScanDataTable = dynamic")
    expect(source).not.toContain('owner="organization-detail-violations"')
    expect(source).not.toContain("loading: () => <DataTableSkeleton")
  })

  it("keeps scheduled scans loading on the resolved scheduled scan table geometry", () => {
    expect(source).toContain("function OrganizationDetailScheduledScansTableLoadingState")
    expect(source).toContain('data-slot="organization-detail-scheduled-scans-table-loading-state"')
    expect(source).not.toContain("function OrganizationDetailScheduledScansTableSkeleton")
    expect(source).not.toContain('data-slot="organization-detail-scheduled-scans-table-skeleton"')
    expect(source).toContain("data={[]}")
    expect(source).toContain("columns={state.scheduledColumns}")
    expect(source).toContain("searchAfter={<OrganizationScheduledScanFilters state={state} />}")
    expect(source).toContain("loadingRowCount={getDataTableSkeletonRowCount(state.scheduledPagination.pageSize)}")
    expect(source).toContain("showQuickFilters={false}")
    expect(source).toContain("enableRowSelection={false}")
    expect(source).not.toContain("columns={6}")
    expect(source).not.toContain("toolbarButtonCount={2}")
  })

  it("uses one shared panel for scheduled status and workflow filters", () => {
    expect(source).toContain("DataTableFacetPanel")
    expect(source).toContain("type DataTableFacetPanelFacet")
    expect(source).toContain('id: "status"')
    expect(source).toContain('id: "workflow"')
    expect(source).toContain("values: state.scheduledStatusFilter")
    expect(source).toContain("values: state.scheduledWorkflowFilter")
    expect(source).toContain("activeCount={state.scheduledStatusFilter.length + state.scheduledWorkflowFilter.length}")
    expect(source).not.toContain("<Select")
  })

  it("shows unfiltered target and scheduled totals in the detail tabs", () => {
    expect(source).toContain("TabsCountBadge")
    expect(source).toContain("state.organizationSummary.totalTargets")
    expect(source).toContain("state.organizationSummary.scheduledTotal")
  })

  it("does not show a dialog-shaped loading state before opening the create scheduled scan sheet", () => {
    expect(source).not.toContain('owner="organization-detail-create-dialog-loading"')
    expect(source).toContain('owner="organization-detail-edit-dialog-loading"')
  })

  it("uses shared structural action placeholders for non-table loading actions", () => {
    expect(source).toContain('from "@/components/shared/loading/action-skeleton"')
    expect(source).toContain("<ActionSkeleton")
    expect(source).not.toContain("function ToolbarButtonSkeleton(")
  })

  it("keeps the detail skeleton aligned to resolved content without legacy route headings", () => {
    expect(source).toContain("OrganizationDetailSummaryRegion")
    expect(source).toContain("OrganizationDetailPrimaryTableRegion")
    expect(source).toContain('getLoadingStructureSlotAttributes("organization-detail-summary")')
    expect(source).toContain('getLoadingStructureSlotAttributes("organization-detail-primary-table")')
    expect(source).not.toContain('owner: "organization-detail-view-loading-state"')
    expect(source).toContain('data-slot="organization-detail-view-loading-state"')
    expect(source).not.toContain('owner: "organization-detail-view-skeleton"')
    expect(source).not.toContain('data-slot="organization-detail-view-skeleton"')
    expect(source).toContain('data-slot="organization-detail-targets-table-loading-state"')
    expect(source).not.toContain('data-slot="organization-detail-targets-table-skeleton"')
    expect(source).toContain('summarySurface = "page"')
    expect(source).toContain('summarySurface === "page" && "py-5"')
    expect(source).toContain("surface === \"drawer\"")
    expect(source).toContain("<TargetsDataTable")
    expect(source).toContain("columns={state.targetColumns}")
    expect(source).toContain("typeFilter={state.typeFilter}")
    expect(source).toContain("const targetStableSurfaceRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)")
    expect(source).toContain("loadingRowCount={targetStableSurfaceRowCount}")
    expect(source.match(/stableSurfaceRowCount=\{targetStableSurfaceRowCount\}/g)).toHaveLength(2)
    expect(source).not.toContain('from "@/components/shared/loading/search-toolbar-skeleton"')
    expect(source).not.toContain('from "@/components/shared/loading/select-shell-skeleton"')
    expect(source).not.toContain('from "@/components/shared/loading/compact-pagination-skeleton"')
    expect(source).not.toContain("<SearchToolbarSkeleton")
    expect(source).not.toContain("<SelectShellSkeleton")
    expect(source).not.toContain("<CompactPaginationSkeleton")
    expect(source).not.toContain("<colgroup>")
    expect(source).not.toContain('<span className="text-muted-foreground">/</span>')
    expect(source).not.toContain("border border-input bg-background")
  })

  it("extends the resolved header and summary structures for detail skeleton geometry", () => {
    expect(source).toContain("state: OrganizationDetailViewState")
    expect(source).toContain("<OrganizationDetailHeader")
    expect(source).toContain("state={state}")
    expect(source).toContain("previewDescription={previewDescription}")
    expect(source).toContain("<OrganizationSummaryStrip")
    expect(source).toContain("surface={summarySurface}")
    expect(source).toContain("loading />")
    expect(source).toContain("mt-1 line-clamp-2 min-h-12 md:min-h-0")
    expect(source).toContain("function InlineSlotLoadingState")
    expect(source).not.toContain("function InlineSlotSkeleton")
    expect(source).not.toContain("ORGANIZATION_DETAIL_SUMMARY_METRIC_COUNT")
    expect(source).not.toContain('Skeleton className="h-8 w-48')
    expect(source).not.toContain('Skeleton className="mt-2 h-8 w-16')
  })

  it("uses resolved localized date text to preserve narrow summary geometry while loading", () => {
    expect(source).toContain("const createdAtText = organization")
    expect(source).toContain("const updatedAtText = organization")
    expect(source).toContain("text={createdAtText}")
    expect(source).toContain("text={updatedAtText}")
    expect(source).toContain("const latestScanValue = summary?.latestScanAt")
    expect(source).toContain("loadingText={latestScanValue}")
    expect(source).toContain("loadingTextUsesContentWidth")
    expect(source).toContain('loadingTextUsesContentWidth ? undefined : loadingClassName ?? "w-10"')
    expect(source).not.toContain('loadingText="2024/12/27 22:30:00"')
    expect(source).not.toContain('className="w-36"')
  })

  it("keeps the line-clamped description as direct text while loading", () => {
    const headerSource = source.slice(
      source.indexOf("function OrganizationDetailHeader"),
      source.indexOf("function OrganizationMetric")
    )
    const descriptionSource = headerSource.match(
      /<p\s+aria-hidden=\{loading \|\| undefined\}[\s\S]*?<\/p>/
    )?.[0]

    expect(descriptionSource).toContain("relative mt-1 line-clamp-2 min-h-12 md:min-h-0")
    expect(descriptionSource).toContain('loading && "select-none text-transparent"')
    expect(descriptionSource).toMatch(/>\s*\{organizationDescription\}\s*\{loading \? \(/)
    expect(descriptionSource).toContain('className="loading-skeleton !absolute inset-0 block rounded-md"')
    expect(descriptionSource).not.toContain("InlineSlotLoadingState")
  })

  it("keeps the drawer target table loading state on the resolved table state", () => {
    expect(source).toContain("ORGANIZATION_DETAIL_FALLBACK_TARGETS_TABLE_COLUMN_COUNT")
    expect(source).toContain("ORGANIZATION_DETAIL_FALLBACK_TARGETS_TABLE_TOOLBAR_BUTTON_COUNT")
    expect(source).toContain("function OrganizationDetailTargetsTableLoadingState")
    expect(source).not.toContain("function OrganizationDetailTargetsTableSkeleton")
    expect(source).toContain("state?: OrganizationDetailViewState")
    expect(source).toContain("pagination={state.pagination}")
    expect(source).toContain("cursorPaginationSummary={state.cursorPaginationSummary}")
    expect(source).toContain("paginationNavigation={state.paginationNavigation}")
    expect(source).toContain("onPaginationChange={state.handlePaginationChange}")
    expect(source).toContain("onBulkDelete={state.handleBulkDelete}")
    expect(source).not.toContain("columns={4}")
    expect(source).not.toContain("toolbarButtonCount={1}")
  })

  it("keeps localized metric labels from widening the organization summary strip", () => {
    expect(source).toContain('type OrganizationDetailSummarySurface = "page" | "drawer"')
    expect(source).toContain("grid min-w-0 overflow-hidden rounded-md border bg-card sm:grid-cols-2 lg:grid-cols-4")
    expect(source).toContain("grid min-w-0 overflow-hidden rounded-md border bg-card sm:grid-cols-2 xl:grid-cols-8")
    expect(source).toContain("flex min-w-0 items-center gap-2")
    expect(source).toContain("shrink-0 text-muted-foreground")
    expect(source).toContain("min-w-0 max-w-full truncate")
    expect(source).toContain("sm:col-span-2 lg:col-span-2")
    expect(source).toContain("sm:col-span-2 xl:col-span-2")
  })

  it("omits the enabled scheduled-scan metric from the summary strip", () => {
    expect(source).not.toContain("const EnabledScanIcon")
    expect(source).not.toContain('label={state.tOrg("detail.metrics.scheduledEnabled")}')
    expect(source).not.toContain("value={summary.scheduledEnabled}")
  })

  it("keeps common translation keys used by detail error states in both locales", () => {
    const commonKeys = Array.from(source.matchAll(/tCommon\("([^"]+)"\)/g), ([, key]) => key)
    const missingKeys = commonKeys.flatMap((key) => {
      const missingLocales = [
        ["en", enMessages.common],
        ["zh", zhMessages.common],
      ].flatMap(([locale, messages]) => (
        typeof readNestedMessage(messages as Record<string, unknown>, key) === "string" ? [] : [`${locale}:${key}`]
      ))

      return missingLocales
    })

    expect(missingKeys).toEqual([])
  })

  it("keeps bulk target unlink confirmation summary-only by default", () => {
    expect(source).toContain("bulkDeleteDialogOpen")
    expect(source).toContain("bulkUnlinkTargetMessage")
    expect(source).toContain("confirmUnlink")
    expect(source).not.toContain("state.selectedTargets.map")
  })
})
