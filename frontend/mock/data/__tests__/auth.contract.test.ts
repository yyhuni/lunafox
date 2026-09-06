import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/auth.ts"), "utf8")

describe("auth contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from '@/types/auth.types'")
  })
})
