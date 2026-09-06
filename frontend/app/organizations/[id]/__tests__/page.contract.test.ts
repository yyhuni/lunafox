import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/organizations/[id]/page.tsx"), "utf8")

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default async function OrganizationDetailPage")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/organization/organization-detail-view\"")
    expect(source).toContain("params: Promise<{ id: string }>")
    expect(source).toContain("const resolvedParams = await params")
  })

  it("directly imports the route-critical organization detail surface instead of delaying its first-screen skeleton behind a null chunk fallback", () => {
    expect(source).not.toContain("lazyPage(")
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toContain("loading: () => null")
    expect(source).not.toContain("OrganizationDetailViewLoadingState")
    expect(source).not.toContain("PageSectionSkeleton")
  })
})
