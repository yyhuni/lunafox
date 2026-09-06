import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/separator.tsx"), "utf8")

describe("separator contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses Base UI Separator as the shared primitive baseline", () => {
    expect(source).toContain('from "@base-ui/react/separator"')
    expect(source).toContain("SeparatorPrimitive")
    expect(source).not.toContain("@radix-ui/react-separator")
  })
})
