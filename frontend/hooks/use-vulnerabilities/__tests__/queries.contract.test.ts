import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-vulnerabilities/queries.ts"), "utf8")

describe("queries contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useAllVulnerabilities")
    expect(source).toContain("from \"@tanstack/react-query\"")
  })

  it("uses canonical vulnerability list params and keeps page tokens in query keys", () => {
    expect(source).toContain("pageSize: params.pageSize ?? 10")
    expect(source).toContain("pageToken: params?.pageToken")
    expect(source).toContain("filter: params?.filter")
    expect(source).toContain("orderBy: params?.orderBy")
    expect(source).toContain("totalSize")
    expect(source).toContain("nextPageToken")
    expect(source).not.toContain("page: 1")
    expect(source).not.toContain("severity: params")
    expect(source).not.toContain("isReviewed: params")
  })

  it("exposes scope-aware vulnerability filter option hooks", () => {
    expect(source).toContain("useGlobalVulnerabilityFilterOptions")
    expect(source).toContain("useTargetVulnerabilityFilterOptions")
    expect(source).toContain("useScanVulnerabilityFilterOptions")
    expect(source).toContain("VulnerabilityService.getGlobalFilterOptions")
    expect(source).toContain("VulnerabilityService.getTargetFilterOptions")
    expect(source).toContain("VulnerabilityService.getScanFilterOptions")
  })
})
