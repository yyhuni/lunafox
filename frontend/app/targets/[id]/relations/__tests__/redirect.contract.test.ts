import { readFileSync } from "node:fs"
import path from "node:path"
import { describe, expect, it } from "vitest"

const source = readFileSync(path.resolve(process.cwd(), "app/targets/[id]/relations/page.tsx"), "utf8")

describe("legacy target relation route", () => {
  it("redirects the former relation list to target websites", () => {
    expect(source).toContain('import { redirect } from "next/navigation"')
    expect(source).toContain('redirect(`/targets/${id}/websites/`)')
    expect(source).not.toContain("WebsiteRelationEvidenceView")
  })
})
