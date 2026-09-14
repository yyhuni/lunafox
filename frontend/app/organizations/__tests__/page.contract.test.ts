import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/organizations/page.tsx"), "utf8")

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default async function OrganizationPage")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/organization/organization-list\"")
  })

  it("directly imports the route-critical list workspace so the page header is not followed by a null-fallback gap", () => {
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toContain("loading: () => null")
  })

  it("keeps the primary list in normal page flow", () => {
    expect(source).toContain("COMPACT_PAGE_SHELL_CLASS")
    expect(source).toContain("COMPACT_CONTENT_GUTTER_CLASS")
    expect(source).toContain("className={COMPACT_CONTENT_GUTTER_CLASS}")
    expect(source).not.toContain("min-h-0")
  })
})
