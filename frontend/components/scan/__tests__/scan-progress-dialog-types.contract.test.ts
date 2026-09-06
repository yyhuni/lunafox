import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scan-progress-dialog-types.ts"), "utf8")

describe("scan-progress-dialog-types contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"@/types/scan.types\"")
  })
})
