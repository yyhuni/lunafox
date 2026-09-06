import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "scripts/build-interaction-matrix.mjs"), "utf8")

describe("build-interaction-matrix contract", () => {
  it("uses shared smoke auth bootstrap helpers", () => {
    expect(source).toContain("from \"node:fs/promises\"")
    expect(source).toContain("from \"../lib/auth-runtime.mjs\"")
    expect(source).toContain("primeSmokeAuthSession")
  })
})
