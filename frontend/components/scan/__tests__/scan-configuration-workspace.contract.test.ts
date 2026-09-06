import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "components/scan/scan-configuration-workspace.tsx"),
  "utf8"
)

describe("scan-configuration-workspace contract", () => {
  it("owns the shared scan configuration header and route-driven primary tabs", () => {
    expect(source).toContain("export function ScanConfigurationWorkspace")
    expect(source).toContain('from "@/components/common/page-header"')
    expect(source).toContain('from "@/components/ui/tabs"')
    expect(source).toContain('<PageHeader code="SCN-02"')
    expect(source).toContain('TabsList variant="content"')
    expect(source).toContain('TabsTrigger value="workflows" variant="content"')
    expect(source).toContain('TabsTrigger value="engines" variant="content"')
    expect(source).toContain('href={scanConfigurationPaths.workflows}')
    expect(source).toContain('href={scanConfigurationPaths.engines}')
  })

  it("keeps the active tab route-owned rather than storing a local selection", () => {
    expect(source).toContain('export type ScanConfigurationTab = "workflows" | "engines"')
    expect(source).toContain("activeTab: ScanConfigurationTab")
    expect(source).toContain("<Tabs value={activeTab}")
    expect(source).not.toContain("useState")
    expect(source).not.toContain("onValueChange")
  })
})
