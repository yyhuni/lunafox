import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/common/bulk-add-urls-dialog-sections.tsx"), "utf8")

describe("bulk-add-urls-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function BulkAddUrlsDialogHeader")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("renders bulk URL validation as blocking errors plus non-blocking advisories", () => {
    expect(source).toContain("BulkLineValidationInput")
    expect(source).toContain("blockingIssueCount")
    expect(source).toContain("advisoryIssueCount")
    expect(source).toContain("duplicateItems")
    expect(source).toContain("mismatchedItems")
  })
})
