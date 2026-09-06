import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scan-progress-dialog-sections.tsx"), "utf8")

describe("scan-progress-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ScanStatusIcon")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/scan/scan-progress-dialog-types\"")
  })

  it("uses shared scan status config instead of local raw var maps", () => {
    expect(source).toContain("from \"@/lib/status-config\"")
    expect(source).not.toContain("const SCAN_STATUS_STYLES")
    expect(source).not.toContain("bg-[var(--warning)]")
  })

  it("keeps failed and cancelled stage glyphs on their canonical status entries", () => {
    expect(source).toContain("semanticIcons.status.failed")
    expect(source).toContain("semanticIcons.status.cancelled")
    expect(source).not.toContain("IconCircleX")
  })
})
