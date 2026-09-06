import { getCoreRowModel, useReactTable } from "@tanstack/react-table"
import { describe, expect, it } from "vitest"
import { renderHook } from "@testing-library/react"
import { readFileSync } from "node:fs"
import path from "node:path"
import type { ScanRecord } from "@/types/scan.types"
import { createScanHistoryColumns } from "@/components/scan/history/scan-history-columns"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-history-columns.tsx"), "utf8")
const tableLayoutSource = readFileSync(
  path.resolve(process.cwd(), "components/scan/history/scan-history-table-layout.ts"),
  "utf8"
)

function sourceBetween(startMarker: string, endMarker: string) {
  const start = source.indexOf(startMarker)
  const end = source.indexOf(endMarker)

  expect(start).toBeGreaterThanOrEqual(0)
  expect(end).toBeGreaterThan(start)

  return source.slice(start, end)
}

describe("scan-history-columns contract", () => {
  it("keeps execution names available through the TanStack row value", () => {
    const scan = {
      id: 9,
      targetId: 1,
      target: { id: 1, name: "test.com", displayName: "test.com", type: "domain" },
      plannedEngineIds: ["engine.lunafox.subdomain_discovery"],
      triggerType: "manual",
      inputSource: "scanSnapshot",
      createdAt: "2026-07-03T07:44:42Z",
      status: "failed",
      progress: 50,
    } satisfies ScanRecord
    const { result } = renderHook(() => {
      const columns = createScanHistoryColumns({
        formatDate: (value) => value,
        handleDelete: () => {},
        handleStop: () => {},
        t: {
          columns: { target: "Target", summary: "Summary", executedEngines: "Engines", triggerType: "Trigger", createdAt: "Created", status: "Status", progress: "Progress" },
          actions: { scanDetail: "Detail", runtimeDetail: "Runtime", openMenu: "Menu", stop: "Stop", stopScanPending: "Stopping", delete: "Delete", selectAll: "Select all", selectRow: "Select row" },
          tooltips: { viewProgress: "Progress" },
          status: { cancelled: "Cancelled", succeeded: "Succeeded", failed: "Failed", pending: "Pending", running: "Running" },
          summary: { subdomains: "Subdomains", websites: "Websites", ipAddresses: "IPs", endpoints: "Endpoints", vulnerabilities: "Vulnerabilities" },
          triggerTypes: { manual: "Manual", scheduled: "Scheduled", ai: "AI" },
        },
        executedEngineNamesByScanId: new Map([[9, ["子域名发现"]]]),
        executedEngineDescriptionsByScanId: new Map([[9, ["通过情报收集、字典爆破和变体生成发现目标域名的子域名"]]]),
      })
      return useReactTable<ScanRecord>({ data: [scan], columns, getCoreRowModel: getCoreRowModel() })
    })

    expect(result.current.getRowModel().rows[0]?.getValue("executedEngines")).toEqual(["子域名发现"])
  })

  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("places the trigger marker before the target and keeps the standardized empty summary badge", () => {
    const targetColumn = sourceBetween('accessorKey: "target"', 'accessorKey: "cachedStats"')

    expect(source).not.toMatch(/import\s+\{[^}]*\bTarget\b[^}]*\}\s+from\s+"@\/components\/icons"/)
    expect(targetColumn).toContain("<ScanTriggerSourceCell")
    expect(targetColumn).toContain("triggerType={row.original.triggerType}")
    expect(targetColumn).toContain("columnLabel={t.columns.triggerType}")
    expect(targetColumn).toContain('className="flex min-w-0 items-center gap-2"')
    expect(source).toContain("data-badge-type=\"empty\"")
    expect(source).toContain("<Badge variant=\"outline\" data-badge-type=\"empty\">")
    expect(source).toContain('useTranslations("scan.history.overview")')
    expect(source).toContain('tOverview("noSummary")')
    expect(targetColumn).not.toContain("TooltipContent")
  })

  it("wraps complete scan summary and engine Badge groups while preserving engine disclosure", () => {
    const summaryColumn = sourceBetween('accessorKey: "cachedStats"', 'id: "executedEngines"')
    const engineColumn = sourceBetween('id: "executedEngines"', 'accessorKey: "createdAt"')

    expect(summaryColumn).toContain("SCAN_HISTORY_SUMMARY_BADGE_LIST_CLASS")
    expect(source).toContain('const SCAN_HISTORY_SUMMARY_BADGE_LIST_CLASS = "flex min-w-0 flex-wrap items-center gap-1"')
    expect(summaryColumn).toContain("data-scan-summary-badges")
    expect(summaryColumn).toContain("data-badge-type={badge.type}")
    expect(summaryColumn).toContain("name: t.summary.subdomains")
    expect(summaryColumn).toContain("name: t.summary.websites")
    expect(summaryColumn).toContain("name: t.summary.ipAddresses")
    expect(summaryColumn).toContain("name: t.summary.endpoints")
    expect(summaryColumn).toContain("name: t.summary.vulnerabilities")
    expect(summaryColumn).not.toContain(" SUB")
    expect(summaryColumn).not.toContain(" WEB")
    expect(summaryColumn).not.toContain(" VULN")
    expect(summaryColumn).not.toContain("ExpandableBadgeList")
    expect(summaryColumn).not.toContain("singleLinePreview")
    expect(summaryColumn).not.toContain("overflow-hidden")
    expect(engineColumn).toContain("SCAN_HISTORY_ENGINE_BADGE_LIST_CLASS")
    expect(engineColumn).toContain("executedEngineNamesByScanId.get(row.id)")
    expect(engineColumn).toContain("executedEngineNamesByScanId.get(row.original.id)")
    expect(engineColumn).toContain("executedEngineDescriptionsByScanId.get(row.original.id)")
    expect(source).toContain('const SCAN_HISTORY_ENGINE_BADGE_LIST_CLASS = "flex min-w-0 flex-wrap items-center gap-1"')
    expect(engineColumn).toContain("data-scan-engine-badges")
    expect(engineColumn).toContain('data-badge-type="engine"')
    expect(engineColumn).toContain("<Tooltip key={`${name}-${index}`}>")
    expect(source).toContain("const SCAN_HISTORY_ENGINE_TOOLTIP_DELAY_MS = 400")
    expect(source).toContain("const SCAN_HISTORY_ENGINE_TOOLTIP_CLOSE_DELAY_MS = 100")
    expect(engineColumn).toContain("<TooltipProvider delay={SCAN_HISTORY_ENGINE_TOOLTIP_DELAY_MS} closeDelay={SCAN_HISTORY_ENGINE_TOOLTIP_CLOSE_DELAY_MS}>")
    expect(engineColumn).toContain("<TooltipTrigger closeOnClick={false} render={<Badge render={<span tabIndex={0} />}")
    expect(engineColumn).toContain("<TooltipContent")
    expect(engineColumn).toContain("render={<span tabIndex={0} />}")
    expect(engineColumn).not.toContain("ExpandableBadgeList")
    expect(engineColumn).not.toContain("singleLinePreview")
    expect(engineColumn).not.toContain("overflow-hidden")
    expect(engineColumn).toContain('"max-w-full shrink-0 truncate"')
  })

  it("omits execution-node fields from the scan history list columns", () => {
    expect(source).not.toContain('accessorKey: "agentName"')
    expect(source).not.toContain("t.columns.agentName")
    expect(source).not.toContain("SingleBadgeCell")
    expect(source).not.toContain('data-badge-type="agent"')
    expect(tableLayoutSource).not.toContain("agentName")
  })

  it("keeps input source out of the scan history list control surface", () => {
    expect(source).not.toContain('accessorKey: "inputSource"')
    expect(source).not.toContain('id: "inputSource"')
  })

  it("keeps terminal diagnostics out of scan-history rows", () => {
    expect(source).not.toContain("diagnostics")
  })

  it("keeps the target column compact inside the shared dense row rhythm", () => {
    const targetColumn = sourceBetween('accessorKey: "target"', 'accessorKey: "cachedStats"')

    expect(source).toContain('from "./scan-history-table-layout"')
    expect(source).toContain("scanHistoryTableColumnLayout")
    expect(targetColumn).toContain("size: scanHistoryTableColumnLayout.target.size")
    expect(targetColumn).toContain("minSize: scanHistoryTableColumnLayout.target.minSize")
    expect(targetColumn).toContain("maxSize: scanHistoryTableColumnLayout.target.maxSize")
    expect(targetColumn).toContain("truncate")
    expect(targetColumn).not.toContain("line-clamp-4")
    expect(targetColumn).not.toContain("break-all")
  })

  it("uses approved compact icon button sizes for row actions", () => {
    expect(source).toContain("DenseRowActionMenu")
    expect(source).toContain("ariaLabel={t.actions.openMenu}")
    expect(source).toContain("size: scanHistoryTableColumnLayout.actions.size")
    expect(source).not.toContain('className="h-8 hover:bg-primary/10 text-primary w-8"')
    expect(source).not.toContain('className="h-8 p-0 w-8"')
  })

  it("routes status clicks through a page-owned action without bubbling to row click", () => {
    const statusColumn = sourceBetween('accessorKey: "status"', '// Progress column removed')

    expect(statusColumn).toContain("handleStatusClick")
    expect(statusColumn).toContain("statusActionLabel")
    expect(statusColumn).toContain("statusClickable")
    expect(statusColumn).toContain("event.stopPropagation()")
    expect(statusColumn).toContain("handleStatusClick?.(row.original)")
    expect(statusColumn).toContain("const runtimeDetailLabel = statusActionLabel ?? t.actions.runtimeDetail")
    expect(statusColumn).toContain("aria-label={runtimeDetailLabel}")
    expect(statusColumn).toContain("data-row-click-exempt")
    expect(statusColumn).toContain("cursor-pointer")
    expect(statusColumn).toContain("hover:bg-muted/40")
    expect(statusColumn).toContain("focus-visible:ring-2")
    expect(statusColumn).toContain("<TooltipProvider")
    expect(statusColumn).toContain("<TooltipTrigger")
    expect(statusColumn).toContain("<TooltipContent>{runtimeDetailLabel}</TooltipContent>")
    expect(statusColumn).not.toContain("ChevronRight")
    expect(statusColumn).toContain("runtimeDetailLabel")
    expect(statusColumn).not.toContain("handleViewProgress")
  })

  it("keeps runtime details out of the row action menu", () => {
    const actionsColumn = sourceBetween('id: "actions"', "  // Filter out target column")

    expect(source).not.toContain("Activity")
    expect(actionsColumn).not.toContain("handleStatusClick ?")
    expect(actionsColumn).not.toContain("handleStatusClick(scan)")
    expect(actionsColumn).not.toContain("{t.actions.runtimeDetail}")
    expect(actionsColumn).toContain("{t.actions.scanDetail}")
  })

  it("exposes only createdAt as a server sortable scan-history column", () => {
    const createdAtColumn = sourceBetween('accessorKey: "createdAt"', 'accessorKey: "status"')
    const statusColumn = sourceBetween('accessorKey: "status"', '// Progress column removed')

    expect(createdAtColumn).toContain('orderBy: "createdAt"')
    expect(createdAtColumn).toContain('serverSortPerformance: "indexed"')
    expect(createdAtColumn).not.toContain("enableSorting: false")
    expect(statusColumn).not.toContain("orderBy")
    expect(statusColumn).not.toContain("serverSortPerformance")
  })

  it("places localized trigger provenance with the target instead of a dedicated column", () => {
    const targetColumn = sourceBetween('accessorKey: "target"', 'accessorKey: "cachedStats"')

    expect(targetColumn).toContain("ScanTriggerSourceCell")
    expect(targetColumn).toContain("t.triggerTypes[row.original.triggerType]")
    expect(source).toContain("semanticIcons.triggerSource[triggerType]")
    expect(source).toContain('role="img"')
    expect(source).toContain("aria-label={accessibleLabel}")
    expect(source).toContain("data-row-click-exempt=\"true\"")
    expect(source).toContain("TooltipContent>{label}</TooltipContent>")
    expect(source).toContain("const triggerSource = hideTargetColumn ?")
    expect(source).not.toContain('accessorKey: "triggerType"')
    expect(tableLayoutSource).not.toContain("triggerType:")
  })

  it("keeps row action icons aligned by using the shared menu icon rhythm", () => {
    const actionsColumn = sourceBetween('id: "actions"', "  // Filter out target column")

    expect(actionsColumn).toContain("<semanticIcons.action.stop />")
    expect(actionsColumn).not.toContain("mr-2")
  })

  it("keeps target compact and gives executed engines the spare-width ownership", () => {
    const targetColumn = sourceBetween('accessorKey: "target"', 'accessorKey: "cachedStats"')
    const summaryColumn = sourceBetween('accessorKey: "cachedStats"', 'id: "executedEngines"')
    const engineColumn = sourceBetween('id: "executedEngines"', 'accessorKey: "createdAt"')

    expect(tableLayoutSource).toContain(
      'target: { key: "target", size: 220, minSize: 160, maxSize: 280 }'
    )
    expect(targetColumn).toContain('widthPolicy: { mode: "flex", flex: 1.1 }')
    expect(source).toContain('accessorKey: "cachedStats"')
    expect(tableLayoutSource).toContain(
      'cachedStats: { key: "cachedStats", size: 240, minSize: 160, maxSize: 320 }'
    )
    expect(summaryColumn).toContain('widthPolicy: { mode: "flex", flex: 1.3 }')
    expect(source).toContain("size: scanHistoryTableColumnLayout.cachedStats.size")
    expect(source).toContain("minSize: scanHistoryTableColumnLayout.cachedStats.minSize")
    expect(source).toContain("maxSize: scanHistoryTableColumnLayout.cachedStats.maxSize")
    expect(source).toContain('id: "executedEngines"')
    expect(engineColumn).toContain("size: scanHistoryTableColumnLayout.executedEngines.size")
    expect(engineColumn).toContain("minSize: scanHistoryTableColumnLayout.executedEngines.minSize")
    expect(engineColumn).toContain("maxSize: scanHistoryTableColumnLayout.executedEngines.maxSize")
    expect(tableLayoutSource).toContain(
      'executedEngines: { key: "executedEngines", size: 220, minSize: 152, maxSize: 280 }'
    )
    expect(engineColumn).toContain('widthPolicy: { mode: "flex", flex: 1, fill: true }')
    expect(engineColumn).toContain("enableResizing: false")
    expect(engineColumn).toContain("enableAutoSize: false")
    expect(source).toContain("enableSorting: false")
    expect(source).toContain('accessorKey: "createdAt"')
    expect(tableLayoutSource).toContain(
      'createdAt: { key: "createdAt", size: 176, minSize: 176, maxSize: 200 }'
    )
    expect(source).toContain("size: scanHistoryTableColumnLayout.createdAt.size")
    expect(source).toContain("minSize: scanHistoryTableColumnLayout.createdAt.minSize")
    expect(source).toContain("maxSize: scanHistoryTableColumnLayout.createdAt.maxSize")
    expect(source).toContain('accessorKey: "status"')
    expect(tableLayoutSource).toContain(
      'status: { key: "status", size: 112, minSize: 108, maxSize: 140 }'
    )
    expect(source).toContain("maxSize: scanHistoryTableColumnLayout.status.maxSize")
  })
})
