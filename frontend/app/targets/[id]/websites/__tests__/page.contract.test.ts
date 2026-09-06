import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/targets/[id]/websites/page.tsx"), "utf8")

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default async function WebSitesPage")
    expect(source).toContain('from "@/components/shared/layout/detail-asset-content-frame"')
    expect(source).toContain("<DetailAssetContentFrame>")
    expect(source).toContain("from \"@/components/websites/website-relation-evidence-view\"")
  })

  it("directly imports the route-critical child workspace instead of leaving a shell-to-workspace blank gap", () => {
    expect(source).not.toContain("lazyPage(")
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toContain("loading: () => null")
  })

  it("always renders the target website evidence workspace", () => {
    expect(source).not.toContain("relation-prototype")
    expect(source).not.toContain("WebsiteRelationEvidenceLab")
    expect(source).toContain("<TargetWebsiteEvidenceView targetId={targetId} />")
  })
})
