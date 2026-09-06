import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "components/shared/layout/detail-asset-content-frame.tsx"),
  "utf8"
)
const readme = readFileSync(
  path.resolve(process.cwd(), "components/shared/layout/README.md"),
  "utf8"
)

describe("detail asset content frame contract", () => {
  it("owns the shared responsive gutter and stable slot", () => {
    expect(source).toContain("export function DetailAssetContentFrame")
    expect(source).toContain("data-slot=\"detail-asset-content-frame\"")
    expect(source).toContain('cn("px-4 pb-4 md:pb-6 lg:px-6", className)')
    expect(source).toContain('React.ComponentProps<"div">')
  })

  it("documents the page and fallback ownership boundary", () => {
    expect(readme).toContain("DetailAssetContentFrame")
    expect(readme).toContain("resolved route page")
    expect(readme).toContain("state-less route fallback")
    expect(readme).toContain("inside a domain `*LoadingState`")
  })
})
