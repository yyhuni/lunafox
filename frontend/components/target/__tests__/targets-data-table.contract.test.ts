import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/target/targets-data-table.tsx"), "utf8")

describe("targets-data-table contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function TargetsDataTable")
    expect(source).toContain("className")
    expect(source).toContain("UnifiedDataTable")
    expect(source).toContain("SimpleSearchToolbar")
    expect(source).toContain("from \"@/components/shared/data-table/target-type-filter-select\"")
  })

  it("opts the target page table toolbar into compact 32px data-table density", () => {
    expect(source).toContain('toolbarDensity="compact"')
    expect(source).toContain('toolbarDensity: "compact"')
    expect(source).toContain("TargetTypeFacetedFilter")
    expect(source).toContain("labels={{")
    expect(source).toContain('all: state.tActions("all")')
    expect(source).toContain('domain: state.tTarget("types.domain")')
    expect(source).toContain('ip: state.tTarget("types.ip")')
    expect(source).toContain('cidr: state.tTarget("types.cidr")')
    expect(source).not.toContain('toolbarDensity="standard"')
    expect(source).not.toContain('toolbarDensity: "standard"')
    expect(source).not.toContain('<SelectTrigger size="default"')
  })

  it("wires the target type control as a multi-select faceted filter", () => {
    expect(source).toContain("typeFilter?: TargetType[]")
    expect(source).toContain("onTypeFilterChange?: (value: TargetType[]) => void")
    expect(source).toContain("<TargetTypeFacetedFilter")
    expect(source).toContain('title={state.tColumns("common.type")}')
    expect(source).toContain("values={typeFilter ?? []}")
    expect(source).toContain("onValuesChange={onTypeFilterChange}")
    expect(source).toContain('clearLabel={state.tDataTable("clearFilter")}')
    expect(source).not.toContain("value={typeFilter || \"all\"}")
    expect(source).not.toContain("onValueChange={(value) => onTypeFilterChange(value === \"all\" ? \"\" : value)}")
  })

  it("pins target name and organization as the paired expand columns for stable width allocation", () => {
    expect(source).toContain('expandColumnIds: ["name", "organizations"]')
    expect(source).toContain('columnLayout: "fixed"')
    expect(source).not.toContain('enableAutoColumnSizing: true')
  })

  it("uses the shared comfortable row rhythm for target identity cells", () => {
    expect(source).toContain('rowDensity: "comfortable"')
  })

  it("renders selected-row scan and delete actions through the shared action bar", () => {
    expect(source).toContain("SelectedRowActionBar")
    expect(source).toContain("onBulkInitiateScan")
    expect(source).toContain('label: state.tTooltips("initiateScan")')
    expect(source).toContain('group: "scan"')
    expect(source).toContain('group: "danger"')
    expect(source).toContain("onClick: onBulkDelete")
    expect(source).toContain("onClearSelection")
    expect(source).not.toContain('bulkDeleteLabel: state.tActions("bulkDelete")')
    expect(source).not.toContain('role="toolbar"')
    expect(source).not.toContain('bulkDeleteLabel: state.tActions("delete")')
  })

  it("hides the column visibility control on the target list toolbar", () => {
    expect(source).toContain("showColumnVisibility: false")
  })

  it("uses server sorting mode for the migrated target list", () => {
    expect(source).toContain('sortingMode: "server"')
    expect(source).toContain("sorting?:")
    expect(source).toContain("onSortingChange?:")
    expect(source).toContain("sorting: state.sorting")
    expect(source).toContain("onSortingChange")
    expect(source).not.toContain('sortingMode: "client"')
  })

  it("uses the shared initial-loading presentation for cold list skeletons", () => {
    expect(source).toContain('loadingPresentation: loading ? "initial" : undefined')
  })

  it("does not expose a workspace-fill override to target table callers", () => {
    expect(source).not.toContain("fillAvailableHeight")
  })
})
