import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/resizable.tsx"), "utf8")

describe("resizable contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/icons\"")
    expect(source).not.toContain("from \"lucide-react\"")
  })
})
