import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/targets/[id]/vulnerabilities/page.tsx"), "utf8")

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default async function TargetVulnerabilitiesPage")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/vulnerabilities/vulnerabilities-detail-view\"")
    expect(source).toContain("params: Promise<{ id: string }>")
    expect(source).toContain("const { id } = await params")
  })

  it("directly imports the route-critical child workspace instead of leaving a shell-to-workspace blank gap", () => {
    expect(source).not.toContain("dynamic(")
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toContain("loading: () => null")
    expect(source).toContain("targetId=")
  })
})
