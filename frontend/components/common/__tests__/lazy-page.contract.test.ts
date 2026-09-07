import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/common/lazy-page.tsx"), "utf8")

describe("lazy-page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function lazyPage")
    expect(source).toContain("from \"next/dynamic\"")
    expect(source).toContain("PageSectionSkeleton")
    expect(source).toContain("loadingOwner")
  })

  it("allows routes to pass a structure-matched fallback or explicit null fallback", () => {
    expect(source).toContain("loadingFallback")
    expect(source).toContain("loadingFallback?: ReactNode | null")
    expect(source).toContain("const hasCustomFallback = arguments.length >= 3")
    expect(source).toContain("hasCustomFallback ? loadingFallback : <PageSectionSkeleton")
  })
})
