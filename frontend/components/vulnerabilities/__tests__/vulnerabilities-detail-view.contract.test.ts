import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/vulnerabilities/vulnerabilities-detail-view.tsx"), "utf8")

describe("vulnerabilities-detail-view contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function VulnerabilitiesDetailView")
    expect(source).toContain("from \"./vulnerabilities-detail-view-sections\"")
  })

  it("opens vulnerability row details through the shared workbench drawer", () => {
    expect(source).toContain("useState<Vulnerability | null>")
    expect(source).toContain("useCallback((vulnerability: Vulnerability)")
    expect(source).toContain("useVulnerabilitiesDetailViewState({ scanId, targetId, websiteScope })")
    expect(source).toContain("onRowClick={handleSelectVulnerability}")
    expect(source).toContain("<VulnerabilityDetailDrawer")
    expect(source).toContain("open={Boolean(activeVulnerability)}")
    expect(source).toContain("onOpenChange={handleDetailOpenChange}")
  })

  it("routes initial loading through the real vulnerabilities table owner", () => {
    expect(source).toContain("ContentHandoff")
    expect(source).toContain("const loadingRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)")
    expect(source).toContain("state={state}")
    expect(source).toContain("rowCount={loadingRowCount}")
    expect(source).toContain("hideToolbar={hideToolbar}")
    expect(source).not.toContain("skeleton={<VulnerabilitiesDetailViewLoadingState rowCount={loadingRowCount} />}")
  })
})
