import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const handlerSource = readFileSync(path.resolve(process.cwd(), "mock/handlers/index.ts"), "utf8")
const indexSource = readFileSync(path.resolve(process.cwd(), "mock/index.ts"), "utf8")

describe("ip address filter options mock contract", () => {
  it("exports port option aggregation through the shared mock barrel", () => {
    expect(indexSource).toContain("getMockPortOptions")
  })

  it("handles target and scan hostPort filter option routes", () => {
    expect(handlerSource).toContain("getMockPortOptions")
    expect(handlerSource).toContain("targetHostPortFilterOptionsMatch")
    expect(handlerSource).toContain("scanHostPortFilterOptionsMatch")
  })
})
