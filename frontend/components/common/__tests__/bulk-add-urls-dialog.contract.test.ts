import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/common/bulk-add-urls-dialog.tsx"), "utf8")

describe("bulk-add-urls-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function BulkAddUrlsDialog")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })
})
