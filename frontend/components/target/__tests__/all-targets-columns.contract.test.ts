import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/target/all-targets-columns.tsx"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "components/target/all-targets-table-layout.ts"), "utf8")

describe("all-targets-columns contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses semantic icons for repeated target row actions", () => {
    expect(source).toContain('import { semanticIcons } from "@/components/icons"')
    expect(source).toContain("semanticIcons.action.view")
    expect(source).toContain("semanticIcons.action.run")
    expect(source).toContain("semanticIcons.action.schedule")
    expect(source).toContain("semanticIcons.action.delete")
  })

  it("keeps the frequent scan action visible beside the overflow menu", () => {
    expect(source).toContain("DenseRowActionMenu")
    expect(source).toContain("DenseRowActionButton")
    expect(source).toContain("leadingActions")
    expect(source).toContain("label={t.tooltips.initiateScan}")
    expect(source).toContain("ariaLabel={t.actions.openMenu}")
    expect(source).toContain("<DropdownMenuItem onClick={onView}>")
    expect(source).toContain("<DropdownMenuItem onClick={onScheduleScan}>")
    expect(source).not.toContain("<DropdownMenuItem onClick={onInitiateScan}>")
    expect(source).toContain('from "./all-targets-table-layout"')
    expect(source).toContain('header: () => <span className="sr-only">{t.columns.actions}</span>')
    expect(source).not.toContain("stickyRight: allTargetsTableColumnLayout.actions.stickyRight")
    expect(source).toContain("size: allTargetsTableColumnLayout.actions.size")
    expect(source).not.toContain("group-hover:opacity-100 items-center opacity-0")
    expect(source).not.toContain('className="w-48"')
  })

  it("keeps the row action column in the normal table flow", () => {
    expect(layoutSource).toContain('actions: { key: "actions", size: 88, minSize: 88, maxSize: 88 }')
    expect(layoutSource).not.toContain("stickyRight")
  })

  it("does not hide the copy affordance from touch users", () => {
    expect(source).toContain('import { QuietCopyButton } from "@/components/shared/data-table/row-actions"')
    expect(source).toContain("<QuietCopyButton")
    expect(source).toContain("copyLabel={t.tooltips.clickToCopy}")
    expect(source).toContain("copiedLabel={t.tooltips.copied}")
    expect(source).not.toContain("async function copyToClipboard")
  })

  it("keeps hover feedback quiet for the inline copy affordance", () => {
    expect(source).toContain("QuietCopyButton")
    expect(source).not.toContain("hover:bg-accent")
  })

  it("does not add a hover tooltip to the dense-table inline copy affordance", () => {
    expect(source).toContain("QuietCopyButton")
    expect(source).not.toContain("<TooltipProvider delayDuration={300}>")
    expect(source).not.toContain('<TooltipContent side="top">')
  })

  it("keeps the target-name cell on the dense table text baseline", () => {
    expect(source).toContain("items-center")
    expect(source).toContain("textRole.tableCellPrimary")
    expect(source).toContain("truncate")
    expect(source).not.toContain("items-start")
    expect(source).not.toContain("textRole.bodyStrong")
    expect(source).not.toContain("break-all")
    expect(source).not.toContain("whitespace-normal")
  })

  it("keeps the copy affordance inline with the target-name text", () => {
    expect(source).toContain("max-w-full")
    expect(source).toContain("block min-w-0 truncate")
    expect(source).not.toContain('className="group flex min-w-0 flex-1 items-center gap-1"')
  })

  it("uses the shared Tooltip for the target-name detail affordance", () => {
    const cellStart = source.indexOf("const TargetNameCell")
    const cellEnd = source.indexOf("const TargetRowActions")
    const cellBlock = source.slice(cellStart, cellEnd)

    expect(source).toContain('from "@/components/ui/tooltip"')
    expect(cellBlock).toContain("<Link")
    expect(cellBlock).toContain("hover:underline")
    expect(cellBlock).toContain("<Tooltip>")
    expect(cellBlock).toContain("TooltipTrigger")
    expect(cellBlock).toContain("TooltipContent")
    expect(cellBlock).toContain("targetDetails")
  })

  it("selects the target subtype icon from the canonical semantic map", () => {
    expect(source).toContain("semanticIcons.concept[targetType]")
    expect(source).toContain("targetType={row.original.type}")
    expect(source).toContain('role="img" aria-label={targetTypeLabel}')
  })

  it("keeps target identity icons lightweight and legible while their row is selected", () => {
    expect(source).toContain("group-data-[state=selected]:text-foreground")
    expect(source).toContain("size-4 shrink-0 text-muted-foreground")
    expect(source).not.toContain("radius-surface")
    expect(source).not.toContain("size-9")
    expect(source).not.toContain("group-data-[state=selected]:border-border")
    expect(source).not.toContain("group-data-[state=selected]:bg-primary")
    expect(source).not.toContain("group-data-[state=selected]:text-primary")
  })

  it("gives timestamp columns a wider stable width band than the primary expand column minimum", () => {
    expect(source).toContain('accessorKey: "name"')
    expect(source).toContain("minSize: allTargetsTableColumnLayout.name.minSize")
    expect(source).toContain('accessorKey: "createdAt"')
    expect(source).toContain("size: allTargetsTableColumnLayout.createdAt.size")
    expect(source).toContain("minSize: allTargetsTableColumnLayout.createdAt.minSize")
    expect(source).toContain("maxSize: allTargetsTableColumnLayout.createdAt.maxSize")
    expect(source).toContain('accessorKey: "lastScannedAt"')
    expect(source).toContain("TimestampCell")
    expect(source).toContain('className="truncate"')
  })

  it("uses a single-line organization preview with an explicit remainder disclosure", () => {
    expect(source).toContain("<ExpandableBadgeList")
    expect(source).toContain("maxVisible={3}")
    expect(source).toContain("singleLinePreview")
    expect(source).not.toContain("wrapPreview")
  })

  it("declares only backend-supported target columns as server-sortable", () => {
    expect(source).toContain('orderBy: "displayName"')
    expect(source).toContain('orderBy: "createdAt"')
    expect(source).toContain('orderBy: "lastScannedAt"')
    expect(source).toContain('firstSortDirection: "asc"')
    expect(source).toContain('firstSortDirection: "desc"')
    expect(source).toContain("serverSortPerformance")
    expect(source).toContain("idx_target_name_id_active")
    expect(source).toContain("idx_target_created_at_id_active")
    expect(source).toContain("idx_target_last_scanned_at_desc_id_active")
    expect(source).toContain("idx_target_last_scanned_at_asc_id_active")
    expect(source).toContain('id: "select"')
    expect(source).toContain('id: "actions"')
    expect(source).toContain('accessorKey: "organizations"')
    expect(source).toContain("enableSorting: false")
  })
})
