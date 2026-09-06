import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/tools/wordlists/page.tsx"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "components/tools/wordlists-page-layout.ts"), "utf8")

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default async function WordlistsPage")
    expect(source).toContain("from \"@/components/tools/wordlists-page\"")
    expect(source).toContain("from \"@/components/common/page-header\"")
    expect(source).toContain("getTranslations(\"pages.tools.wordlists\")")
    expect(source).toContain('code="WDL-01"')
    expect(source).toContain('description={t("description")}')
    expect(source).toContain("WORDLISTS_PAGE_SHELL_CLASS")
    expect(source).toContain("WORDLISTS_CONTENT_SHELL_CLASS")
    expect(layoutSource).toContain('WORDLISTS_PAGE_SHELL_CLASS =\n  "flex h-full min-h-0 flex-1 flex-col gap-4 py-4 md:gap-6 md:py-6"')
    expect(layoutSource).toContain('WORDLISTS_WORKSPACE_HANDOFF_CLASS = "flex h-full min-h-0 flex-1 flex-col"')
    expect(layoutSource).toContain('WORDLISTS_WORKSPACE_SURFACE_CLASS =\n  "flex h-full min-h-0 flex-1 flex-col overflow-hidden"')
  })

  it("directly imports the route-critical wordlist workspace instead of hiding its first-screen owner behind a null chunk fallback", () => {
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toContain("loading: () => null")
    expect(source).not.toContain("lazyPage(")
    expect(source).not.toContain("from \"@/components/common/lazy-page\"")
    expect(source).not.toContain("WordlistsPageSkeleton")
  })
})
