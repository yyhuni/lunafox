import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/common/bulk-line-validation-input.tsx"), "utf8")

describe("bulk-line-validation-input contract", () => {
  it("owns the shared line-numbered validation field shell", () => {
    expect(source).toContain("export function BulkLineValidationInput")
    expect(source).toContain("LineNumberedTextarea")
    expect(source).toContain("lineNumberedTextareaResponsiveViewportClassName")
    expect(source).toContain("lineHighlights")
    expect(source).toContain("blockingIssueCount")
    expect(source).toContain("advisoryIssueCount")
    expect(source).toContain("labelClassName?: string")
    expect(source).toContain("fillHeight?: boolean")
    expect(source).toContain("showEmptySummary?: boolean")
    expect(source).toContain("showEmptySummary = true")
    expect(source).toContain("showSuccessSummary?: boolean")
    expect(source).toContain("showSuccessSummary = true")
    expect(source).toContain("lineNumberedTextareaFillViewportClassName")
    expect(source).toContain('fillHeight ? "flex h-full min-h-0 flex-col" : "grid"')
    expect(source).toContain("className={labelClassName}")
    expect(source).toContain("showEmptySummary && !validationResult")
  })

  it("keeps the issue details disclosure button hover background transparent", () => {
    expect(source).toContain("collapseDetails")
    expect(source).toContain("expandDetails")
    expect(source).toContain("hover:bg-transparent")
    expect(source).toContain("dark:hover:bg-transparent")
  })
})
