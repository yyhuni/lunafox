import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const handlerSource = readFileSync(path.resolve(process.cwd(), "mock/handlers/index.ts"), "utf8")
const indexSource = readFileSync(path.resolve(process.cwd(), "mock/index.ts"), "utf8")

describe("website filter options mock contract", () => {
  it("exports website filter option aggregation through the shared mock barrel", () => {
    expect(indexSource).toContain("getMockWebsiteFilterOptions")
  })

  it("handles target and scan website filter option routes", () => {
    expect(handlerSource).toContain("getMockWebsiteFilterOptions")
    expect(handlerSource).toContain("targetWebsiteFilterOptionsMatch")
    expect(handlerSource).toContain("scanWebsiteFilterOptionsMatch")
  })
})
