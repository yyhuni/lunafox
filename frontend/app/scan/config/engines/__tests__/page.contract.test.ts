import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "app/scan/config/engines/page.tsx"),
  "utf8"
)

describe("scan configuration engines page contract", () => {
  it("mounts the engine catalog inside the shared workspace with the engine tab active", () => {
    expect(source).toContain('from "@/components/scan/scan-configuration-workspace"')
    expect(source).toContain('from "@/components/tools/engines/engine-installation-page"')
    expect(source).toContain('<ScanConfigurationWorkspace activeTab="engines">')
    expect(source).toContain("<EngineInstallationPage embedded />")
    expect(source).not.toContain("<PageHeader")
  })
})
