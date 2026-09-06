import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/tools/config/tool-card.tsx"), "utf8")

describe("tool-card contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ToolCard")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/ui/card\"")
  })
})
