import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/utils.ts"), "utf8")

describe("utils contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function cn")
    expect(source).toContain("from \"clsx\"")
  })
})
