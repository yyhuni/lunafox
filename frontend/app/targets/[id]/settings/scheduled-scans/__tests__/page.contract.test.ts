import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "app/targets/[id]/settings/scheduled-scans/page.tsx"),
  "utf8"
)

describe("scheduled scans page contract", () => {
  it("directly mounts the target scheduled scan workspace", () => {
    expect(source).toContain("export default async function TargetScheduledScansPage")
    expect(source).toContain("from \"@/components/target/target-settings\"")
    expect(source).toContain('section="scheduled-scans"')
    expect(source).toContain("className=\"flex min-h-0 flex-1 flex-col px-4 lg:px-6\"")
  })

  it("does not add a route-level dynamic blank fallback", () => {
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toContain("loading: () => null")
  })
})
