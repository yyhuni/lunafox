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

  it("keeps the repeated tool grid on the compact section rhythm", () => {
    expect(source).toContain('className="flex flex-col gap-3"')
    expect(source).toContain('className="gap-3 grid grid-cols-1 lg:grid-cols-3 md:grid-cols-2 xl:grid-cols-4"')
  })
})
