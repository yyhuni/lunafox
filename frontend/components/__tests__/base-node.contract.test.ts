import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/base-node.tsx"), "utf8")

describe("base-node contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function BaseNode")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })
})
