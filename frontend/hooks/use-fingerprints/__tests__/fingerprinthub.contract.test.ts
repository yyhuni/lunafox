import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-fingerprints/fingerprinthub.ts"), "utf8")

describe("fingerprinthub contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"@/hooks/_shared/fingerprint-hooks\"")
  })
})
