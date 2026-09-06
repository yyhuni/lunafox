import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/form.tsx"), "utf8")

describe("form contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses the shared native label wrapper for form labels", () => {
    expect(source).toContain('import { Label } from "@/components/ui/label"')
    expect(source).toContain("React.ComponentProps<typeof Label>")
    expect(source).not.toContain("@radix-ui/react-label")
    expect(source).not.toContain("LabelPrimitive")
  })

  it("uses project-owned polymorphic composition for form controls", () => {
    expect(source).toContain('from "@/components/ui/polymorphic"')
    expect(source).not.toContain("@radix-ui/react-slot")
  })
})
