import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/skeleton.tsx"), "utf8")

describe("skeleton contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"@/lib/utils\"")
  })

  it("keeps skeleton geometry decorative and motion-owned", () => {
    expect(source).toContain("data-slot=\"skeleton\"")
    expect(source).toContain("aria-hidden=\"true\"")
    expect(source).toContain("loading-skeleton")
  })
})
