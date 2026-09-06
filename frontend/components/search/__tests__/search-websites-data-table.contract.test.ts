import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/search/search-websites-data-table.tsx"), "utf8")
const sections = readFileSync(path.resolve(process.cwd(), "components/search/search-page-sections.tsx"), "utf8")

describe("search websites table contract", () => {
  it("reuses the target website evidence-list table owner", () => {
    expect(source).toContain("RelationEvidenceListFrame")
    expect(source).toContain("SearchWebsitesDataTableLoadingState")
    expect(source).toContain("WebsiteRelationEvidenceLoadingState")
    expect(source).toContain("rowCount={rowCount}")
    expect(source).toContain("WebsiteRelationEvidenceRow")
    expect(source).toContain("RelationScreenshotPreviewDialog")
    expect(source).toContain('targetId={website.target}')
    expect(source).toContain("useSearchWebsitesResultModel")
    expect(source).not.toContain("WebSitesDataTable")
    expect(source).not.toContain("useWebSiteTableColumns")
  })

  it("keeps global search read-only while retaining export, selection, and the empty screenshot state", () => {
    expect(source).toContain("buildExportOptions")
    expect(source).toContain("exportItems(websites, \"all\")")
    expect(source).toContain("exportItems(selectedRows, \"selected\")")
    expect(source).toContain("model.exportOptions")
    expect(source).toContain("<Checkbox")
    expect(source).toContain("toWebsiteEvidence")
    expect(source).toContain('tDataTable("noResults")')
    expect(source).not.toContain("onBulkAdd")
    expect(source).not.toContain("onBulkDelete")
  })

  it("removes the duplicate result search while keeping export in the page toolbar", () => {
    expect(source).toContain("toolbar={null}")
    expect(source).not.toContain("RelationEvidenceToolbar")
    expect(source).not.toContain("SimpleSearchToolbar")
    expect(sections).toContain("<SearchWebsiteExportMenu")
    expect(sections).toContain("model={websiteResultModel}")
  })

  it("replaces only the Website result table and moves Search pagination into its evidence-list footer", () => {
    expect(sections).toContain("<SearchWebsitesDataTable")
    expect(sections).toContain("<SearchWebsitesDataTableLoadingState")
    expect(sections).toContain("rowCount={websiteLoadingRowCount}")
    expect(sections).toContain("model={websiteResultModel}")
    expect(sections).toContain("pagination={(")
    expect(sections).toContain("const pagination = (")
    expect(sections).toContain("websiteLoadingRowCount={state.pageSize}")
    expect(sections).toContain('state.assetType !== "website"')
    expect(sections).toContain("<SearchResultsToolbarRegion>")
    expect(sections).toContain("<AssetSearchBar state={state}")
    expect(sections).toContain("<SearchResultsPaginationRegion>")
    expect(sections).not.toContain("state.data.results.map((result) => <SearchResultCard")
  })
})
