import { readFileSync } from "node:fs"
import path from "node:path"
import { describe, expect, it } from "vitest"

function readSource(relativePath: string) {
  return readFileSync(path.resolve(process.cwd(), relativePath), "utf8")
}

const evidenceSource = readSource("components/websites/website-relation-evidence-view.tsx")
const targetWebsiteRouteSource = readSource("app/targets/[id]/websites/page.tsx")
const scanWebsiteRouteSource = readSource("app/scan/history/[id]/websites/page.tsx")
const targetLayoutSource = readSource("app/targets/[id]/layout.tsx")
const websiteStateSource = readSource("components/websites/websites-view-state.ts")

describe("target website evidence workspace contract", () => {
  it("makes the evidence workspace the target website view while preserving the scan table", () => {
    expect(targetWebsiteRouteSource).toContain("<TargetWebsiteEvidenceView targetId={targetId} />")
    expect(targetWebsiteRouteSource).not.toContain("<WebSitesView")
    expect(scanWebsiteRouteSource).toContain("<WebSitesView scanId={scanId} />")
  })

  it("keeps website collection state authoritative for controls and rendered records", () => {
    expect(evidenceSource).toContain("useWebSitesViewState({ targetId")
    expect(evidenceSource).toContain("state.websites.map")
    expect(evidenceSource).toContain("state.commitFilterSearch")
    expect(evidenceSource).toContain("state.handlePaginationChange")
    expect(evidenceSource).toContain("state.handleSelectionChange")
  })

  it("keeps selected website deletion in the shared action bar and existing confirmation flow", () => {
    expect(evidenceSource).toContain("<SelectedRowActionBar")
    expect(evidenceSource).toContain('onClick: () => state.setDeleteDialogOpen(true)')
    expect(evidenceSource).toContain("<WebSitesManagementOverlays state={state} />")
    expect(evidenceSource).not.toContain("onBulkDelete={selectedCount > 0")
  })

  it("orders website, response, and screenshot evidence consistently", () => {
    const listFrameSource = evidenceSource.slice(
      evidenceSource.indexOf("function RelationEvidenceListFrame"),
      evidenceSource.indexOf("function RelationEvidenceToolbar")
    )
    const rowSource = evidenceSource.slice(
      evidenceSource.indexOf("function WebsiteRelationEvidenceRow"),
      evidenceSource.indexOf("export function WebsiteScreenshot")
    )

    expect(listFrameSource.indexOf('t("columns.website")')).toBeLessThan(listFrameSource.indexOf('t("columns.response")'))
    expect(listFrameSource.indexOf('t("columns.response")')).toBeLessThan(listFrameSource.indexOf('t("columns.screenshot")'))
    expect(rowSource.indexOf("<WebsiteIdentity")).toBeLessThan(rowSource.indexOf("<WebsiteResponseEvidence"))
    expect(rowSource.indexOf("<WebsiteResponseEvidence")).toBeLessThan(rowSource.indexOf("<WebsiteScreenshot"))
  })

  it("keeps pagination outside the bordered records surface", () => {
    const listFrameSource = evidenceSource.slice(
      evidenceSource.indexOf("function RelationEvidenceListFrame"),
      evidenceSource.indexOf("function RelationEvidenceToolbar")
    )

    expect(listFrameSource).toContain('data-slot="data-table-pagination-surface"')
    expect(listFrameSource).not.toContain('className="border-t border-border px-4 py-3"')
  })

  it("reuses the existing website export callbacks without adding screenshot export", () => {
    expect(evidenceSource).toContain("buildExportOptions")
    expect(evidenceSource).toContain("onExportAll: state.handleExportAll")
    expect(evidenceSource).toContain("onExportSelected: state.handleExportSelected")
    expect(websiteStateSource).toContain("exportWebsites()")
    expect(websiteStateSource).toContain("-websites-selected-")
    expect(evidenceSource).not.toMatch(/zip|manifest|exportScreenshot/i)
  })

  it("removes the relation primary tab and treats website detail as an Assets route", () => {
    expect(targetLayoutSource).not.toContain('<TabsTrigger value="relations"')
    expect(targetLayoutSource).not.toContain('relations: `${basePath}/relations/`')
    expect(targetLayoutSource).toContain('const isWebsiteDetailRoute = /^\\/targets\\/[^/]+\\/websites\\/[^/]+/.test(pathname);')
    expect(targetLayoutSource).toContain("showSecondaryNav = primaryTab === \"settings\" || (primaryTab === \"assets\" && !isWebsiteDetailRoute)")
  })
})
