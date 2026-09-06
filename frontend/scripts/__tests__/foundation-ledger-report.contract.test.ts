import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const scriptPath = path.resolve(process.cwd(), "scripts/report-foundation-ledger.mjs")

describe("foundation ledger debt report", () => {
  it("exists and groups debt by check, status, owner, and route group", () => {
    expect(existsSync(scriptPath)).toBe(true)

    const source = readFileSync(scriptPath, "utf8")

    expect(source).toContain("foundation-exceptions.json")
    expect(source).toContain("groupByCheck")
    expect(source).toContain("groupByStatus")
    expect(source).toContain("groupByOwner")
    expect(source).toContain("groupByRouteGroup")
    expect(source).toContain("remove")
    expect(source).toContain("deferred")
  })
})
