import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/error-code-map.ts"), "utf8")

describe("error-code-map contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function getErrorI18nKey")
  })
})
