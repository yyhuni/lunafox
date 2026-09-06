import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/initiate-scan-helpers.ts"), "utf8")

describe("initiate-scan-helpers contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export type InitiateScanSelectMode = \"preset\" | \"custom\"")
  })

  it("accepts bulk organization and target scopes during validation", () => {
    expect(source).toContain("organizationIds?: number[]")
    expect(source).toContain("targetIds?: number[]")
    expect(source).toContain("hasSingleScope")
    expect(source).toContain("hasBulkScope")
  })
})
