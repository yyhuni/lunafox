import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/tools/page.tsx"), "utf8")

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default async function ToolsPage")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/ui/button\"")
  })

  it("uses approved Button sizing for card actions", () => {
    expect(source).not.toContain('<Button className="w-full"')
    expect(source).not.toContain('<Button disabled className="w-full"')
  })

  it("does not expose the retired Space Mapping workflow", () => {
    expect(source).not.toContain("space-mapping")
    expect(source).not.toContain("target.spaceImport")
    expect(source).not.toContain("spaceMapping")
  })
})
