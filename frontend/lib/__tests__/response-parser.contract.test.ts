import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/response-parser.ts"), "utf8")

describe("response-parser contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function isErrorResponse")
  })
})
