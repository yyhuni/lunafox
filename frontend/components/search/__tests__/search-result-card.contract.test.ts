import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/search/search-result-card.tsx"), "utf8")

describe("search-result-card contract", () => {
  it("renders Website evidence through the shared response panel and existing card owner", () => {
    expect(source).toContain("export function SearchResultCard")
    expect(source).toContain('import { ResponseEvidencePanel } from "@/components/shared/response-evidence"')
    expect(source).toContain("<ResponseEvidencePanel")
    expect(source).toContain("location={result.location}")
    expect(source).toContain('emptyLabel: t("noResponseContent")')
    expect(source).toContain("responseHeaders")
    expect(source).toContain("responseBody")
    expect(source).not.toContain("showMetadata")
    expect(source).not.toContain("TabsContent")
    expect(source).toContain("ExpandableTagList")
  })

  it("routes HTTP status codes through the shared badge owner", () => {
    expect(source).toContain("HttpStatusBadge")
    expect(source).toContain("statusCode={result.statusCode}")
    expect(source).not.toContain("function getStatusVariant")
    expect(source).not.toContain("status >= 200")
  })

  it("does not couple global search cards to vulnerability summaries", () => {
    expect(source).not.toContain("vulnerabilit")
    expect(source).not.toContain("getSeverityVariant")
  })

  it("does not animate the technologies clamp with max-height", () => {
    expect(source).not.toContain("transition-[max-height]")
  })
})
