import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/subdomains/bulk-add-subdomains-dialog.tsx"), "utf8")

describe("bulk-add-subdomains-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function BulkAddSubdomainsDialog")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })
})
