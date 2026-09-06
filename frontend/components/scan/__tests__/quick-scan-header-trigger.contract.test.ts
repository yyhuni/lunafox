import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/quick-scan-header-trigger.tsx"), "utf8")
const readme = readFileSync(path.resolve(process.cwd(), "components/scan/README.md"), "utf8")

describe("quick scan header trigger contract", () => {
  it("owns the global header trigger and dialog composition", () => {
    expect(source).toContain("export function QuickScanHeaderTrigger")
    expect(source).toContain("QuickScanDialog")
    expect(source).toContain('data-slot="quick-scan-trigger"')
    expect(source).toContain('variant="ghost"')
    expect(source).toContain('size="icon-sm"')
    expect(source).toContain('aria-label={t("quickScan")}')
    expect(source).toContain("IconZap")
    expect(source).not.toContain("PageActionButton")
  })

  it("documents the independent global-header boundary", () => {
    expect(readme).toContain("QuickScanHeaderTrigger")
    expect(readme).toContain("does not inline a quick-scan dialog trigger")
  })
})
