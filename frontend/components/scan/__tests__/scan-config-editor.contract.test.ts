import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scan-config-editor.tsx"), "utf8")

describe("scan-config-editor contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ScanConfigEditor")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/scan/scan-config-editor-sections\"")
  })

  it("can hide its visual label when embedded under an external section title", () => {
    expect(source).toContain("showLabel?: boolean")
    expect(source).toContain("showLabel = true")
    expect(source).toContain("showLabel={showLabel}")
  })
})
