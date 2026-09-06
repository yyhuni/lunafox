import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/fingerprint.types.ts"), "utf8")

describe("fingerprint.types contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("/**")
  })
})
