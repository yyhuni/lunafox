import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/website.types.ts"), "utf8")

describe("website.types contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("/**")
  })

  it("models canonical Website identity, optional screenshot summaries, and a read-only scope", () => {
    expect(source).toContain("resourceName?: string")
    expect(source).toContain("screenshot?: WebsiteScreenshotSummary")
    expect(source).toContain("export interface WebsiteAssetScope")
    expect(source).not.toContain("WebsiteRelationEvidence")
  })
})
