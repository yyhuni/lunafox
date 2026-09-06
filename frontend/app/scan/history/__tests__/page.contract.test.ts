import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/scan/history/page.tsx"), "utf8")

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default async function ScanHistoryPage")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/scan/history/scan-history-list\"")
  })

  it("renders the shared page header above scan history content", () => {
    expect(source).toContain("from \"@/components/common/page-header\"")
    expect(source).toContain("<PageHeader")
    expect(source).toContain('code="SCH-HIS"')
    expect(source).toContain('title={tScan("history.title")}')
    expect(source).toContain('description={tScan("history.description")}')
  })
})
