import { describe, expect, it } from "vitest"
import { readdirSync, readFileSync, statSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/faceted-filter.tsx"), "utf8")

function readTsxSources(directory: string): Array<{ file: string; source: string }> {
  return readdirSync(directory).flatMap((entry) => {
    const file = path.join(directory, entry)
    const stats = statSync(file)
    if (stats.isDirectory()) return readTsxSources(file)
    if (!file.endsWith(".tsx")) return []
    return [{ file, source: readFileSync(file, "utf8") }]
  })
}

describe("data-table faceted filter contract", () => {
  it("owns the shared dashed multi-select filter button used by business-list toolbars", () => {
    expect(source).toContain("export function DataTableFacetedFilter")
    expect(source).toContain("values: TValue[]")
    expect(source).toContain("onValuesChange: (values: TValue[]) => void")
    expect(source).toContain("options: Array<DataTableFacetedFilterOption<TValue>>")
    expect(source).toContain("count?: number")
    expect(source).toContain('export type DataTableFacetedFilterContentSize = "default" | "wide"')
    expect(source).toContain("facetedFilterContentSizeClassNames")
    expect(source).toContain('default: "w-48"')
    expect(source).toContain('wide: "w-56"')
    expect(source).toContain("contentSize?: DataTableFacetedFilterContentSize")
    expect(source).toContain('contentSize = "default"')
    expect(source).toContain("searchPlaceholder?: string")
    expect(source).toContain("toggleFacetedFilterValue")
    expect(source).not.toContain("getFacetedFilterTriggerWidthOptions")
    expect(source).toContain("<Popover open={open} onOpenChange={handleOpenChange}>")
    expect(source).toContain('<Button variant="outline" size="sm" className="border-dashed"')
    expect(source).toContain('className="radius-round flex size-4 items-center justify-center border border-foreground/80"')
    expect(source).toContain('{selectedOptions.length > 0 ? (<>')
    expect(source).not.toContain('aria-hidden="true" className="invisible')
    expect(source).not.toContain('triggerWidthOptions.map')
    expect(source).not.toContain("selectedOptions.length === 0 && \"invisible\"")
    expect(source).toContain('import { TabsCountBadge } from "@/components/ui/tabs"')
    expect(source).toContain('<TabsCountBadge className="font-normal lg:hidden">')
    expect(source).toContain('<span className="hidden gap-1 lg:flex">')
    expect(source).toContain('selectedOptions.length > 1')
    expect(source).toContain('selectedOptions.map((option) => (')
    expect(source).toContain('{option.label}')
    expect(source).toContain('className="max-w-28 justify-start overflow-hidden font-normal"')
    expect(source).toContain('className="block min-w-0 truncate text-left"')
    const triggerSource = source.slice(source.indexOf("<PopoverTrigger"), source.indexOf("</PopoverTrigger>"))
    expect(triggerSource).toContain("selectedOptions.length")
    expect(triggerSource).toContain("option.label")
    expect(triggerSource).toContain("selectedOptions.map")
    expect(source).not.toContain("selectedSummary")
    expect(source).toContain('<PopoverContent minWidth="content" className={cn(facetedFilterContentSizeClassNames[contentSize], "overflow-hidden p-0", contentClassName)} align="start">')
    expect(source).not.toContain("!min-w-0")
    expect(source).toContain("<Command>")
    expect(source).toContain('import { useTranslations } from "next-intl"')
    expect(source).toContain('const tDataTable = useTranslations("dataTable")')
    expect(source).toContain("CommandInput")
    expect(source).toContain('<CommandInput placeholder={searchPlaceholder ?? tDataTable("searchFilter", { title })}/>')
    expect(source).toContain('<CommandList className="max-h-64">')
    expect(source).toContain("value={option.value}")
    expect(source).toContain("keywords={[option.label]}")
    expect(source).toContain('<div className="shrink-0">')
    expect(source).toContain("{clearLabel}")
  })

  it("keeps direct faceted selections as a draft until the user applies them", () => {
    const directFilterSource = source.slice(
      source.indexOf("export function DataTableFacetedFilter"),
      source.indexOf("export function DataTableFacetedFilterGroup")
    )

    expect(directFilterSource).toContain('const [draftValues, setDraftValues] = useState<TValue[]>(values)')
    expect(directFilterSource).toContain("const hasDraftChanges = !haveSameFacetValues(values, draftValues)")
    expect(directFilterSource).toContain("const handleOpenChange")
    expect(directFilterSource).toContain("setDraftValues(values)")
    expect(directFilterSource).toContain("const resetDraft = () => setDraftValues([])")
    expect(directFilterSource).toContain("const applyDraft")
    expect(directFilterSource).toContain("onValuesChange(draftValues)")
    expect(directFilterSource).toContain("onOpenChange={handleOpenChange}")
    expect(directFilterSource).toContain("values={draftValues}")
    expect(directFilterSource).toContain("onValuesChange={setDraftValues}")
    expect(directFilterSource).toContain("showClear={false}")
    expect(directFilterSource).toContain("onClick={resetDraft}")
    expect(directFilterSource).toContain("disabled={!hasDraftChanges}")
    expect(directFilterSource).toContain("onClick={applyDraft}")
  })

  it("keeps reset ownership on the shared faceted-filter group instead of each filter trigger", () => {
    expect(source).toContain('import { Check, Filter, Plus, XIcon } from "@/components/icons"')
    expect(source).toContain("export function DataTableFacetedFilterGroup")
    expect(source).toContain("hasSelectedValues: boolean")
    expect(source).toContain("onReset: () => void")
    expect(source).toContain("const onClearFilters = () => onValuesChange([])")
    expect(source).toContain('const tActions = useTranslations("common.actions")')
    expect(source).toContain('{hasSelectedValues ? (')
    expect(source).toContain('variant="ghost"')
    expect(source).toContain('size="sm"')
    expect(source).toContain('aria-label={tActions("reset")}')
    expect(source).toContain('{tActions("reset")}')
    expect(source).toContain('<XIcon className="size-4"/>')

    const filterSource = source.slice(source.indexOf("export function DataTableFacetedFilter"), source.indexOf("export function DataTableFacetedFilterGroup"))
    expect(filterSource).not.toContain('aria-label={tActions("reset")}')
    expect(filterSource).not.toContain('<XIcon className="size-4"/>')
  })

  it("renders optional option counts as right-aligned numeric affordances", () => {
    expect(source).toContain('typeof option.count === "number"')
    expect(source).toContain('"ml-auto shrink-0 tabular-nums text-muted-foreground"')
    expect(source).toContain("option.count.toLocaleString()")
    expect(source).toContain('import { textRole } from "@/lib/typography"')
  })

  it("uses the shared interaction accent for selected option markers", () => {
    expect(source).toContain('"border-interaction-accent bg-interaction-accent text-interaction-accent-foreground"')
    expect(source).not.toContain('"border-primary bg-primary text-primary-foreground"')
  })

  it("owns the aggregate filter trigger and compact single-facet layout", () => {
    expect(source).toContain("export function DataTableFacetPanel")
    expect(source).toContain("export type DataTableFacetPanelFacet")
    expect(source).toContain("facets: DataTableFacetPanelFacet[]")
    expect(source).toContain("activeCount: number")
    expect(source).toContain("<Filter className=\"size-4\"/>")
    expect(source).toContain("const [activeFacetId, setActiveFacetId] = useState")
    expect(source).toContain("const activeFacet = draftFacets.find")
    expect(source).toContain("DataTableFacetedFilterOptions")
    expect(source).toContain("const isSingleFacet = draftFacets.length === 1")
    expect(source).toContain('isSingleFacet ? "w-72" : "w-80 sm:w-96"')
    expect(source).toContain('className="flex h-80 overflow-hidden"')
    expect(source).toContain('data-facet-layout={isSingleFacet ? "single" : "multiple"}')
    expect(source).toContain('!isSingleFacet ? (<div className="flex h-full w-40 shrink-0 flex-col border-r">')
    expect(source).toContain('{tDataTable("filterDimensions")}')
    expect(source).toContain('className={cn("border-b px-3 py-2", textRole.bodyStrong)}')
    expect(source).toContain('layout="between"')
    expect(source).toContain('layout="fullWidth"')
    expect(source).not.toContain("DataTableActiveFilter")
    expect(source).not.toContain("DataTableActiveFilterList")
    expect(source).not.toContain("onRemove: () => void")
    expect(source).not.toContain('size="chip-icon"')
  })

  it("keeps grouped facet selections as a draft until the user applies them", () => {
    expect(source).toContain('const [draftValues, setDraftValues] = useState')
    expect(source).toContain("const handleOpenChange")
    expect(source).toContain("const resetDraft")
    expect(source).toContain("const applyDraft")
    expect(source).toContain("onOpenChange={handleOpenChange}")
    expect(source).toContain("onValuesChange: (values: string[]) => setDraftValues")
    expect(source).toContain("{draftFacets.map((facet) => (")
    expect(source).toContain('disabled={!hasDraftChanges}')
    expect(source).toContain('{tActions("reset")}')
    expect(source).toContain('{tDataTable("applyFilter")}')
  })

  it("keeps faceted-filter popover widths on the shared size API instead of route width utilities", () => {
    const routeWidthOverrides = readTsxSources(path.resolve(process.cwd(), "components"))
      .filter((item) => item.source.includes("DataTableFacetedFilter"))
      .filter((item) => /contentClassName="w-(48|52|56)"/.test(item.source))

    expect(routeWidthOverrides).toEqual([])
  })
})
