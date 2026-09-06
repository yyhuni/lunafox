import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const drawerPath = path.resolve(process.cwd(), "components/websites/website-detail-drawer.tsx")
const source = existsSync(drawerPath) ? readFileSync(drawerPath, "utf8") : ""

describe("website-detail-drawer contract", () => {
  it("uses the shared read-only detail drawer with a single inspection page", () => {
    expect(source).toContain("DetailDrawer")
    expect(source).not.toContain("DetailDrawerTabs")
    expect(source).toContain("WebsiteDrawerSection")
    expect(source).toContain('label={tColumns("website.technologies")}')
    expect(source).toContain('label={tColumns("website.responseHeaders")}')
    expect(source).toContain('label={tColumns("website.responseBody")}')
  })

  it("contains response payload overflow inside a readable code region", () => {
    expect(source).toContain("overflow-auto")
    expect(source).toContain("font-mono")
    expect(source).toContain("whitespace-pre-wrap")
  })
})
