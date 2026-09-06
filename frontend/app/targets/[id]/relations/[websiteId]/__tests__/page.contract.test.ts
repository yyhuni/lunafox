import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/targets/[id]/relations/[websiteId]/[[...section]]/page.tsx"), "utf8")

describe("website relation detail page contract", () => {
  it("redirects legacy detail sections to canonical website routes", () => {
    expect(source).toContain('import { redirect } from "next/navigation"')
    expect(source).toContain('const sectionPath = section?.length ? `${section.join("/")}/` : ""')
    expect(source).toContain('redirect(`/targets/${id}/websites/${websiteId}/${sectionPath}`)')
    expect(source).not.toContain("WebsiteRelationDetailView")
  })
})
