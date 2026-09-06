import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/websites/websites-view-state.ts"), "utf8")

describe("websites-view-state contract", () => {
  it("defaults secondary location, host, and response payload columns to hidden while keeping visibility user-controlled", () => {
    expect(source).toContain("DEFAULT_WEBSITE_COLUMN_VISIBILITY")
    expect(source).toContain("useWebSiteTableColumns")
    expect(source).toContain("host: false")
    expect(source).toContain("location: false")
    expect(source).toContain("responseBody: false")
    expect(source).toContain("responseHeaders: false")
    expect(source).toContain("columnVisibility")
    expect(source).toContain("setColumnVisibility")
  })
})
