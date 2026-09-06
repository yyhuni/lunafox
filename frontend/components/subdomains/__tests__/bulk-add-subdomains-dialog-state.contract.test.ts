import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/subdomains/bulk-add-subdomains-dialog-state.ts"), "utf8")

describe("bulk-add-subdomains-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useBulkAddSubdomainsDialogState")
    expect(source).toContain("from \"react\"")
  })

  it("uses line-preserving validation and exposes blocking versus advisory issue counts", () => {
    expect(source).toContain("SubdomainValidator.parseLines")
    expect(source).toContain("blockingIssueCount")
    expect(source).toContain("advisoryIssueCount")
    expect(source).toContain("duplicateItems")
    expect(source).toContain("lineIssues")
  })
})
