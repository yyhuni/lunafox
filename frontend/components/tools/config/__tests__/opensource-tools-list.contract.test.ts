import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/tools/config/opensource-tools-list.tsx"), "utf8")

describe("opensource-tools-list contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function OpensourceToolsList")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })
})
