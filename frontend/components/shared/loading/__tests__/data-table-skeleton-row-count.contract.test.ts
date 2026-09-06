import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const root = process.cwd()

function readSource(relativePath: string) {
  return readFileSync(path.resolve(root, relativePath), "utf8")
}

const pageSizedTableLoadingOwners = [
  "components/scan/history/scan-history-list.tsx",
  "components/overview/overview-scan-history.tsx",
  "components/overview/overview-scheduled-scans.tsx",
  "components/organization/organization-list.tsx",
  "components/organization/organization-detail-view-sections.tsx",
  "components/vulnerabilities/vulnerabilities-vertical-view.tsx",
  "components/vulnerabilities/vulnerabilities-detail-view.tsx",
  "components/websites/websites-view.tsx",
  "components/subdomains/subdomains-detail-view.tsx",
  "components/ip-addresses/ip-addresses-view.tsx",
  "components/directories/directories-view.tsx",
  "components/endpoints/endpoints-detail-view.tsx",
  "components/organization/targets/targets-detail-view-sections.tsx",
  "components/target/all-targets-detail-view.tsx",
  "components/target/target-settings.tsx",
  "components/target/target-settings-sections.tsx",
]

const fingerprintTableLoadingOwners = [
  "components/fingerprints/fingerprinthub-fingerprint-view.tsx",
]

describe("data table skeleton row-count contract", () => {
  it("exports a shared first-screen row cap helper", () => {
    const source = readSource("components/shared/loading/data-table-skeleton.tsx")

    expect(source).toContain("export const DATA_TABLE_SKELETON_MAX_ROWS = 10")
    expect(source).toContain("export function getDataTableSkeletonRowCount")
    expect(source).toContain("throw new Error")
  })

  it.each(pageSizedTableLoadingOwners)(
    "%s derives initial table skeleton rows from active page size",
    (relativePath) => {
      const source = readSource(relativePath)

      expect(source).toContain("getDataTableSkeletonRowCount")
      expect(source).not.toMatch(/rows=\{(?:3|4|6|8)\}/)
    }
  )

  it("centralizes fingerprint table loading row count in the shared workspace", () => {
    const workspace = readSource("components/fingerprints/fingerprint-library-workspace.tsx")

    expect(workspace).toContain("getDataTableSkeletonRowCount(queryState.pageSize)")
    expect(workspace).toContain("loadingRowCount,")
    expect(workspace).toContain("stableSurfaceRowCount: loadingRowCount")

    for (const relativePath of fingerprintTableLoadingOwners) {
      const source = readSource(relativePath)

      expect(source).toContain("FingerprintLibraryWorkspace")
      expect(source).not.toContain("<FingerprintDataTableSkeleton")
      expect(source).not.toMatch(/rows=\{(?:3|4|6|8)\}/)
    }
  })
})
