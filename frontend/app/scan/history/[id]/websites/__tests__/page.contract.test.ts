import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/scan/history/[id]/websites/page.tsx"), "utf8")

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default async function ScanWebSitesPage")
    expect(source).toContain('from "@/components/shared/layout/detail-asset-content-frame"')
    expect(source).toContain("<DetailAssetContentFrame>")
    expect(source).toContain("from \"@/components/websites/websites-view\"")
  })

  it("directly imports the route-critical child workspace instead of leaving a shell-to-workspace blank gap", () => {
    expect(source).not.toContain("lazyPage(")
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toContain("loading: () => null")
  })
})
