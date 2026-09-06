import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/settings/blacklist/page.tsx"), "utf8")

describe("page contract", () => {
  it("keeps the route wrapper thin and lets content own the blacklist loading skeleton", () => {
    expect(source).toContain("export default function GlobalBlacklistPage")
    expect(source).toContain("<GlobalBlacklistPageContent />")
    expect(source).toContain("import GlobalBlacklistPageContent from \"./content\"")
    expect(source).not.toContain("getTranslations")
    expect(source).not.toContain("lazyPage(")
    expect(source).not.toContain("dynamic(")
    expect(source).not.toContain("BlacklistSettingsSkeleton")
  })

  it("keeps loading ownership inside blacklist content", () => {
    expect(source).not.toContain("RouteSegmentLoadingOwner")
    expect(source).not.toContain("RouteFallback")
  })
})
