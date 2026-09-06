import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/smart-filter.ts"), "utf8")

describe("smart-filter contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function getTranslatedFields")
  })
})
