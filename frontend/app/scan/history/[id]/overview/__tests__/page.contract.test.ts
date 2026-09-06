import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/scan/history/[id]/overview/page.tsx"), "utf8")

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default async function ScanOverviewPage")
    expect(source).toContain("className")
    expect(source).toContain('from "@/components/scan/history/scan-overview"')
  })

  it("keeps the scan overview route-critical chunk directly imported so the shell skeleton owns first paint", () => {
    expect(source).toContain("import { ScanOverview }")
    expect(source).not.toContain("lazyPage")
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toContain("scan-overview-chunk")
    expect(source).not.toContain("ScanOverviewLoadingState")
    expect(source).not.toContain("PageSectionSkeleton")
  })
})
