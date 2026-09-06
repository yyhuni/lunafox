import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/targets/[id]/overview/page.tsx"), "utf8")

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default async function TargetOverviewPage")
    expect(source).toContain("className")
    expect(source).toContain('from "@/components/target/target-overview"')
  })

  it("keeps the overview route-critical chunk directly imported so the shell skeleton owns first paint", () => {
    expect(source).toContain("import { TargetOverview }")
    expect(source).not.toContain("lazyPage")
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toContain("target-overview-chunk")
    expect(source).not.toContain("TargetOverviewLoadingState")
    expect(source).not.toContain("PageSectionSkeleton")
  })
})
