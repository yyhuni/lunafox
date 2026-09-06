import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/subdomains/bulk-add-subdomains-dialog-sections.tsx"), "utf8")

describe("bulk-add-subdomains-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function BulkAddSubdomainsHeader")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("renders subdomain validation as blocking errors plus non-blocking advisories", () => {
    expect(source).toContain("blockingIssueCount")
    expect(source).toContain("advisoryIssueCount")
    expect(source).toContain("LineNumberedTextarea")
    expect(source).toContain("lineNumberedTextareaResponsiveViewportClassName")
    expect(source).toContain("viewportClassName={lineNumberedTextareaResponsiveViewportClassName}")
    expect(source).toContain("lineHighlights")
    expect(source).toContain("duplicateItems")
  })
})
