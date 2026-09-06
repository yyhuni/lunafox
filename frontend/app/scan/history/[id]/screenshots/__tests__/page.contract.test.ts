import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/scan/history/[id]/screenshots/page.tsx"), "utf8")

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default async function ScanScreenshotsPage")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/screenshots/screenshots-gallery\"")
  })

  it("directly imports the route-critical child workspace instead of leaving a shell-to-workspace blank gap", () => {
    expect(source).not.toContain("lazyPage(")
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toContain("loading: () => null")
  })
})
