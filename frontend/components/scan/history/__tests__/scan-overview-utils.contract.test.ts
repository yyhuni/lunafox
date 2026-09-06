import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-overview-utils.ts"), "utf8")

describe("scan-overview-utils contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function formatStageDuration")
    expect(source).toContain("from \"@/components/icons\"")
  })

  it("reuses shared scan status config instead of owning a local style map", () => {
    expect(source).toContain("from \"@/lib/status-config\"")
    expect(source).not.toContain("const SCAN_STATUS_STYLES")
    expect(source).not.toContain("bg-[var(--warning)]")
  })
})
