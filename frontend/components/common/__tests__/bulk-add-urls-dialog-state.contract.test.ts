import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/common/bulk-add-urls-dialog-state.ts"), "utf8")

describe("bulk-add-urls-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useBulkAddUrlsDialogState")
    expect(source).toContain("from \"react\"")
  })

  it("uses line-preserving URL validation and exposes blocking versus advisory issue counts", () => {
    expect(source).toContain("URLValidator.parseLines")
    expect(source).toContain("blockingIssueCount")
    expect(source).toContain("advisoryIssueCount")
    expect(source).toContain("duplicateItems")
    expect(source).toContain("mismatchedItems")
    expect(source).toContain("lineIssues")
    expect(source).toContain("hasNonWhitespaceInput")
    expect(source).not.toContain("inputText.trim()")
  })
})
