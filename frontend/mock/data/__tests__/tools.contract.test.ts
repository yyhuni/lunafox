import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/tools.ts"), "utf8")

describe("tools contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function getMockTools")
    expect(source).toContain("from '@/types/tool.types'")
  })
})
