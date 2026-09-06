import { readFileSync } from "node:fs"
import path from "node:path"
import { describe, expect, it } from "vitest"

const businessErrorOwners = [
  "components/settings/blacklist/blacklist-settings-workspace.tsx",
  "app/targets/[id]/layout.tsx",
  "components/directories/directories-view.tsx",
  "components/endpoints/endpoints-detail-view.tsx",
  "components/ip-addresses/ip-addresses-view.tsx",
  "components/organization/organization-detail-view.tsx",
  "components/organization/organization-list.tsx",
  "components/organization/targets/targets-detail-view.tsx",
  "components/overview/overview-data-table-sections.tsx",
  "components/overview/overview-scan-history.tsx",
  "components/scan/history/scan-history-list.tsx",
  "components/scan/history/scan-overview.tsx",
  "components/screenshots/screenshots-gallery.tsx",
  "components/search/search-page-sections.tsx",
  "components/settings/database-health/database-health-view.tsx",
  "components/subdomains/subdomains-detail-view.tsx",
  "components/target/all-targets-detail-view.tsx",
  "components/target/target-overview.tsx",
  "components/target/target-settings.tsx",
  "components/tools/config/custom-tools-list.tsx",
  "components/tools/config/opensource-tools-list.tsx",
  "components/tools/engines/engine-installation-page.tsx",
  "components/tools/nuclei-poc-catalog-page.tsx",
  "components/websites/websites-view.tsx",
] as const

const contextualCollectionOwners = [
  "components/settings/blacklist/blacklist-settings-workspace.tsx",
  "components/directories/directories-view.tsx",
  "components/endpoints/endpoints-detail-view.tsx",
  "components/ip-addresses/ip-addresses-view.tsx",
  "components/organization/organization-list.tsx",
  "components/organization/targets/targets-detail-view.tsx",
  "components/overview/overview-data-table-sections.tsx",
  "components/overview/overview-scan-history.tsx",
  "components/scan/history/scan-history-list.tsx",
  "components/scan/history/scan-overview.tsx",
  "components/screenshots/screenshots-gallery.tsx",
  "components/search/search-page-sections.tsx",
  "components/settings/database-health/database-health-view.tsx",
  "components/subdomains/subdomains-detail-view.tsx",
  "components/target/all-targets-detail-view.tsx",
  "components/target/target-settings.tsx",
  "components/tools/config/custom-tools-list.tsx",
  "components/tools/config/opensource-tools-list.tsx",
  "components/tools/engines/engine-installation-page.tsx",
  "components/tools/nuclei-poc-catalog-page.tsx",
  "components/websites/websites-view.tsx",
] as const

const fingerprintWorkspace = "components/fingerprints/fingerprint-library-workspace.tsx"

const fingerprintViews = [
  "components/fingerprints/fingerprinthub-fingerprint-view.tsx",
] as const

const compactSectionOwners = [
  "components/directories/directories-view.tsx",
  "components/endpoints/endpoints-detail-view.tsx",
  "components/ip-addresses/ip-addresses-view.tsx",
  "components/organization/targets/targets-detail-view.tsx",
  "components/overview/overview-data-table-sections.tsx",
  "components/overview/overview-scan-history.tsx",
  "components/scan/history/scan-overview.tsx",
  "components/search/search-page-sections.tsx",
  "components/settings/database-health/database-health-view.tsx",
  "components/subdomains/subdomains-detail-view.tsx",
  "components/target/target-settings.tsx",
  "components/tools/config/custom-tools-list.tsx",
  "components/tools/config/opensource-tools-list.tsx",
  "components/tools/engines/engine-installation-page.tsx",
  "components/tools/nuclei-poc-catalog-page.tsx",
] as const

const removedLocalOwners = [
  ["components/target/all-targets-detail-view-sections.tsx", "AllTargetsDetailViewErrorState"],
  ["components/organization/organization-list-sections.tsx", "OrganizationListErrorState"],
  ["components/organization/targets/targets-detail-view-sections.tsx", "TargetsDetailViewErrorState"],
  ["components/directories/directories-view-sections.tsx", "DirectoriesViewErrorState"],
  ["components/endpoints/endpoints-detail-view-sections.tsx", "EndpointsDetailViewErrorState"],
  ["components/ip-addresses/ip-addresses-view-sections.tsx", "IPAddressesViewErrorState"],
  ["components/subdomains/subdomains-detail-view-sections.tsx", "SubdomainsDetailViewErrorState"],
  ["components/target/target-settings-sections.tsx", "TargetSettingsErrorState"],
] as const

function readSource(relativePath: string) {
  return readFileSync(path.resolve(process.cwd(), relativePath), "utf8")
}

describe("production business error state adoption", () => {
  it.each(businessErrorOwners)("routes %s through the shared semantic owner", (relativePath) => {
    const source = readSource(relativePath)

    expect(source).toContain("AppErrorState")
    expect(source).not.toContain("error.message")
    expect(source).not.toContain("state.error?.message")
    expect(source).not.toContain("Request failed with status code")
  })

  it("centralizes fingerprint list and detail errors in the shared workspace", () => {
    const source = readSource(fingerprintWorkspace)

    expect(source).toContain("AppErrorState")
    expect(source).toContain('notFoundKind: "unexpected-error"')
    expect(source).toContain("onRetry={listQuery.error ? listQuery.refetch : facetOptionsQuery.refetch}")
    expect(source).toContain("onRetry={detailQuery.refetch}")
    expect(source).toContain('variant="section"')

    for (const relativePath of fingerprintViews) {
      expect(readSource(relativePath)).toContain("FingerprintLibraryWorkspace")
    }
  })

  it.each(contextualCollectionOwners)("classifies collection 404 context in %s", (relativePath) => {
    expect(readSource(relativePath)).toContain('notFoundKind: "unexpected-error"')
  })

  it.each(compactSectionOwners)("uses the shared compact presentation in %s", (relativePath) => {
    expect(readSource(relativePath)).toContain('variant="section"')
  })

  it.each(removedLocalOwners)("does not restore %s in %s", (relativePath, ownerName) => {
    expect(readSource(relativePath)).not.toContain(ownerName)
  })
})
