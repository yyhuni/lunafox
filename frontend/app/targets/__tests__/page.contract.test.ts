import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/targets/page.tsx"), "utf8")

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default async function AllTargetsPage")
    expect(source).toContain("from \"@/components/target/all-targets-detail-view\"")
    expect(source).toContain("<AllTargetsDetailView />")
    expect(source).not.toContain("from \"next/dynamic\"")
    expect(source).not.toContain("/tools/space-mapping/")
    expect(source).not.toContain("target.spaceImport")
  })

  it("keeps the top-level target list in normal page flow", () => {
    expect(source).toContain('className="flex flex-col gap-4 py-4 md:gap-6 md:py-6"')
    expect(source).toContain('className="px-4 lg:px-6"')
    expect(source).not.toContain("min-h-0")
    expect(source).not.toContain("h-svh")
    expect(source).not.toContain("h-screen")
    expect(source).not.toContain("min-h-screen")
  })
})
