import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const handoffFiles = [
  "components/screenshots/screenshots-gallery.tsx",
  "components/scan/scheduled/scheduled-scan-page.tsx",
  "components/scan/workflow/scan-workflow-page.tsx",
  "components/settings/notifications/notification-settings-page-content.tsx",
  "components/tools/config/custom-tools-list.tsx",
  "components/tools/config/opensource-tools-list.tsx",
  "components/tools/nuclei-poc-catalog-page.tsx",
  "components/tools/wordlists-page.tsx",
  "components/vulnerabilities/vulnerabilities-detail-view.tsx",
  "components/vulnerabilities/vulnerabilities-vertical-view.tsx",
  "components/settings/blacklist/blacklist-settings-workspace.tsx",
] as const

const fingerprintHandoffViews = [
  "components/fingerprints/fingerprinthub-fingerprint-view.tsx",
] as const

const routeBoundaryHandoffFiles = [
  "app/settings/api-keys/api-keys-settings-workspace.tsx",
  "app/settings/database-health/database-health-workspace.tsx",
  "app/settings/notifications/notification-settings-workspace.tsx",
  "app/settings/support/support-workspace.tsx",
  "app/settings/system-logs/system-logs-workspace.tsx",
  "app/scan/scheduled/scheduled-scan-workspace.tsx",
] as const

const routeBoundaryDynamicFallbackFiles = routeBoundaryHandoffFiles.filter(
  (file) => file !== "app/settings/notifications/notification-settings-workspace.tsx"
)

const detailShellOverviewBoundaryFiles = [
  "app/targets/[id]/layout.tsx",
  "app/scan/history/[id]/layout.tsx",
] as const

const routeHandoffReadinessChildren = [
  "app/settings/api-keys/content.tsx",
  "components/settings/database-health/database-health-view.tsx",
  "components/settings/notifications/notification-settings-page-content.tsx",
  "components/settings/support/support-page-content.tsx",
  "components/settings/system-logs/system-logs-view.tsx",
  "components/scan/scheduled/scheduled-scan-page.tsx",
] as const

const detailShellChildren = [
  "components/target/target-overview.tsx",
  "components/scan/history/scan-overview.tsx",
  "components/websites/websites-view.tsx",
  "components/subdomains/subdomains-detail-view.tsx",
  "components/ip-addresses/ip-addresses-view.tsx",
  "components/endpoints/endpoints-detail-view.tsx",
  "components/directories/directories-view.tsx",
  "components/screenshots/screenshots-gallery.tsx",
  "components/vulnerabilities/vulnerabilities-detail-view.tsx",
  "components/target/target-settings.tsx",
] as const

const detailShellNestedFallbacks = [
  ["components/websites/websites-view-sections.tsx", "WebSitesViewRouteFallback"],
  ["components/subdomains/subdomains-detail-view-sections.tsx", "SubdomainsDetailViewRouteFallback"],
  ["components/ip-addresses/ip-addresses-view-sections.tsx", "IPAddressesViewRouteFallback"],
  ["components/endpoints/endpoints-detail-view-sections.tsx", "EndpointsDetailViewRouteFallback"],
  ["components/directories/directories-view-sections.tsx", "DirectoriesViewRouteFallback"],
  ["components/target/target-settings-sections.tsx", "TargetSettingsRouteFallback"],
] as const

const routeCriticalDirectImportFiles = [
  "app/vulnerabilities/page.tsx",
  "app/vulnerabilities/vulnerabilities-workspace.tsx",
  "app/settings/api-keys/page.tsx",
  "app/settings/database-health/page.tsx",
  "app/settings/notifications/page.tsx",
  "app/settings/support/page.tsx",
  "app/settings/system-logs/page.tsx",
  "app/scan/scheduled/page.tsx",
  "app/targets/[id]/overview/page.tsx",
  "app/scan/history/[id]/overview/page.tsx",
  "app/search/page.tsx",
  "app/organizations/page.tsx",
  "app/organizations/[id]/page.tsx",
  "app/scan/config/workflows/page.tsx",
  "app/scan/config/engines/page.tsx",
  "app/tools/nuclei/page.tsx",
  "app/tools/wordlists/page.tsx",
  "app/tools/fingerprints/fingerprinthub/page.tsx",
  "app/targets/[id]/websites/page.tsx",
  "app/targets/[id]/ip-addresses/page.tsx",
  "app/targets/[id]/subdomains/page.tsx",
  "app/targets/[id]/endpoints/page.tsx",
  "app/targets/[id]/screenshots/page.tsx",
  "app/targets/[id]/settings/page.tsx",
  "app/targets/[id]/settings/scheduled-scans/page.tsx",
  "app/targets/[id]/directories/page.tsx",
  "app/targets/[id]/vulnerabilities/page.tsx",
  "app/scan/history/[id]/websites/page.tsx",
  "app/scan/history/[id]/ip-addresses/page.tsx",
  "app/scan/history/[id]/endpoints/page.tsx",
  "app/scan/history/[id]/directories/page.tsx",
  "app/scan/history/[id]/subdomains/page.tsx",
  "app/scan/history/[id]/screenshots/page.tsx",
  "app/scan/history/[id]/vulnerabilities/page.tsx",
] as const

function readSource(file: string) {
  return readFileSync(path.resolve(process.cwd(), file), "utf8")
}

function initialHandoffSource(_file: string, source: string) {
  return source
}

describe("page loading handoff audit", () => {
  it("documents that ContentHandoff skeleton children must not declare a second same-layer owner", () => {
    const source = readSource("components/shared/loading/README.md")

    expect(source).toContain("a `ContentHandoff` skeleton subtree that declares its own `data-loading-owner`")
    expect(source).toContain("Skeleton components that can be rendered both standalone and inside")
    expect(source).toContain("SHOULD accept an optional `owner`")
  })

  it.each(handoffFiles)("%s uses shared ContentHandoff for initial skeleton replacement", (file) => {
    const source = readSource(file)
    const initialSource = initialHandoffSource(file, source)

    expect(initialSource).toContain("ContentHandoff")
    expect(initialSource).not.toMatch(/if\s*\([^)]*(?:isLoading|loading|isQueryLoading)[^)]*\)\s*\{\s*return\s*\(?\s*<[\s\S]*?(?:Skeleton|LoadingState)/)
  })

  it("centralizes fingerprint initial handoff in the shared workspace", () => {
    const workspace = readSource("components/fingerprints/fingerprint-library-workspace.tsx")

    expect(workspace).toContain("ContentHandoff")
    expect(workspace).toContain("owner={owner}")
    expect(workspace).toContain('layer="workspace"')
    expect(workspace).toContain("isLoading={isInitialLoading}")
    expect(workspace).not.toMatch(/if\s*\([^)]*(?:isLoading|loading|isQueryLoading)[^)]*\)\s*\{\s*return\s*\(?\s*<[\s\S]*?(?:Skeleton|LoadingState)/)

    for (const file of fingerprintHandoffViews) {
      expect(readSource(file)).toContain("FingerprintLibraryWorkspace")
    }
  })

  it.each(routeBoundaryHandoffFiles)("%s routes hidden readiness through the shared route-boundary helper", (file) => {
    const source = readSource(file)

    expect(source).toContain("HiddenReadinessRouteBoundary")
    expect(source).not.toContain("const [isReady, setIsReady] = React.useState(false)")
    expect(source).not.toContain("<ContentHandoff")
    expect(source).toContain("onReady={onReady}")
    expect(source).toContain("deferInitialSkeleton={deferInitialSkeleton}")
  })

  it.each(routeBoundaryDynamicFallbackFiles)("%s keeps its hidden dynamic child fallback", (file) => {
    expect(readSource(file)).toContain("loading: () => null")
  })

  it("notification settings mounts its route-critical content with the boundary", () => {
    const source = readSource("app/settings/notifications/notification-settings-workspace.tsx")

    expect(source).toContain(
      'import NotificationSettingsPageContent from "@/components/settings/notifications/notification-settings-page-content"'
    )
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toContain("loading: () => null")
  })

  it.each([...handoffFiles, ...routeBoundaryHandoffFiles])("%s keeps ContentHandoff skeleton children metadata-free", (file) => {
    const source = readSource(file)

    expect(source).not.toMatch(/skeleton=\{<[^}]+\sowner=/)
  })

  it.each(routeHandoffReadinessChildren)("%s exposes readiness props for a route-owned handoff", (file) => {
    const source = readSource(file)

    expect(source).toContain("onReady?: () => void")
    expect(source).toContain("deferInitialSkeleton?: boolean")
    expect(source).toContain("onReady?.()")
  })

  it.each(detailShellOverviewBoundaryFiles)("%s keeps detail-shell overview handoff on the shared hidden-readiness boundary", (file) => {
    const source = readSource(file)

    expect(source).toContain("HiddenReadinessRouteBoundary")
    expect(source).toContain("DetailShellReadyProvider")
    expect(source).toContain("onReady, deferInitialSkeleton")
  })

  it.each(detailShellChildren)("%s can defer its initial visible skeleton until the detail shell is ready to hand off", (file) => {
    const source = readSource(file)

    expect(source).toContain("useDetailShellReadySignal")
    expect(source).toContain("detailShellReady?.deferInitialSkeleton")
    expect(source).toContain("if (detailShellReady?.deferInitialSkeleton && isInitialLoading) {")
    expect(source).toContain("return null")
  })

  it.each(detailShellNestedFallbacks)("%s keeps %s metadata-free inside the complete detail shell", (file, exportName) => {
    const source = readSource(file)
    const start = source.indexOf(`export function ${exportName}`)
    const remaining = source.slice(start)
    const nextExport = remaining.indexOf("\nexport function ", 1)
    const fallbackSource = nextExport === -1 ? remaining : remaining.slice(0, nextExport)

    expect(start).toBeGreaterThanOrEqual(0)
    expect(fallbackSource).not.toContain('owner="')
  })

  it.each(routeCriticalDirectImportFiles)("%s directly imports route-critical first-screen content", (file) => {
    const source = readSource(file)

    expect(source).not.toContain("lazyPage")
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toMatch(/loading:\s*\(\)\s*=>/)
    expect(source).not.toContain("PageSectionSkeleton")
  })
})
