import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/scan/config/blacklist/page.tsx"), "utf8")

describe("page contract", () => {
  it("keeps the route wrapper inside the scan configuration shell", () => {
    expect(source).toContain("export default function ScanConfigurationBlacklistPage")
    expect(source).toContain("ScanConfigurationWorkspace")
    expect(source).toContain('activeTab="blacklist"')
    expect(source).toContain("<BlacklistSettingsWorkspace />")
    expect(source).not.toContain("redirect(")
    expect(source).not.toContain("getTranslations")
    expect(source).not.toContain("lazyPage(")
    expect(source).not.toContain("dynamic(")
    expect(source).not.toContain("BlacklistSettingsSkeleton")
    expect(source).not.toContain("<PageHeader")
  })

  it("keeps loading ownership inside the blacklist workspace", () => {
    expect(source).not.toContain("RouteSegmentLoadingOwner")
    expect(source).not.toContain("RouteFallback")
  })
})
