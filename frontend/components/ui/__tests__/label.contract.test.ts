import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/label.tsx"), "utf8")

describe("label contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses a native label wrapper because Base UI has no standalone label primitive", () => {
    expect(source).toContain("React.ComponentProps<\"label\">")
    expect(source).toContain("<label")
    expect(source).not.toContain("@radix-ui/react-label")
  })
})
