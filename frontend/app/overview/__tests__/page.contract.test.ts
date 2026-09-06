import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const pageSource = readFileSync(path.resolve(process.cwd(), "app/overview/page.tsx"), "utf8")
const contentSource = readFileSync(path.resolve(process.cwd(), "components/overview/overview-page-content.tsx"), "utf8")

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(pageSource).toContain("export default function Page")
    expect(pageSource).toContain("className")
    expect(pageSource).toContain('from "@/components/overview/overview-page-content"')
    expect(pageSource).not.toContain("bauhaus-overview-header")
  })

  it("keeps the overview dashboard on the shared route-shell rhythm", () => {
    expect(pageSource).toContain('className="flex flex-col gap-4 py-4 md:gap-5 md:pt-5 md:pb-6"')
    expect(pageSource).toContain("<OverviewPageContent />")
    expect(pageSource).not.toContain("px-4 lg:px-6")
    expect(pageSource).not.toContain("AppShellWarmup")
  })

  it("delegates the header and lazy-section loader to the refresh-aware client owner", () => {
    expect(pageSource).not.toContain('from "@/components/overview/overview-page-header"')
    expect(pageSource).not.toContain('from "@/components/overview/overview-sections-loader"')
    expect(contentSource).toContain('from "@/components/overview/overview-page-header"')
    expect(contentSource).toContain('from "@/components/overview/overview-sections-loader"')
    expect(contentSource).toContain("<OverviewPageHeader")
    expect(contentSource).toContain("<OverviewSectionsLoader />")
    expect(contentSource).not.toContain('from "@/components/overview/overview-lazy-sections"')
    expect(contentSource).not.toContain("OverviewLazySections")
  })
})
