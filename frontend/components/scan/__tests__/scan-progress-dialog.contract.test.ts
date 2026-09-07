import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scan-progress-dialog.tsx"), "utf8")

describe("scan-progress-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ScanProgressDialog")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/ui/dialog\"")
  })
})
