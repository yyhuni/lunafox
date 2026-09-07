import { readFileSync } from "node:fs"
import path from "node:path"
import { describe, expect, it } from "vitest"

const source = readFileSync(path.resolve(process.cwd(), "app/targets/[id]/websites/[websiteId]/[[...section]]/page.tsx"), "utf8")

describe("target website detail route", () => {
  it("resolves route-backed website detail sections", () => {
    expect(source).toContain("resolveWebsiteRelationDetailSection(section?.[0])")
    expect(source).toContain("<WebsiteRelationDetailView")
    expect(source).toContain("websiteId={Number(websiteId)}")
  })
})
