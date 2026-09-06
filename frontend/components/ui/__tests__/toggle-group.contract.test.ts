import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/toggle-group.tsx"), "utf8")

describe("toggle-group contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses Base UI primitives while preserving the project toggle-group API", () => {
    expect(source).toContain('from "@base-ui/react/toggle-group"')
    expect(source).toContain('from "@base-ui/react/toggle"')
    expect(source).not.toContain("@radix-ui/react-toggle-group")
    expect(source).toContain("type?: TType")
    expect(source).toContain("value?: string | string[]")
    expect(source).toContain("onValueChange?: (value: string | string[]) => void")
  })

  it("keeps segmented corners under theme control instead of forcing a square middle state", () => {
    expect(source).toContain("radius-surface")
    expect(source).toContain("first:radius-control-left")
    expect(source).toContain("last:radius-control-right")
    expect(source).not.toContain("rounded-none")
  })
})
