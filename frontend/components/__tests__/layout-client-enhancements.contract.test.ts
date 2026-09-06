import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/layout-client-enhancements.tsx"), "utf8")

describe("layout-client-enhancements contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function LayoutClientEnhancements")
    expect(source).toContain("from \"react\"")
    expect(source).toContain("useLocale")
    expect(source).toContain("localeHtmlLang")
    expect(source).toContain("document.documentElement.lang")
    expect(source).toContain("React.useEffect")
    expect(source).not.toContain("from \"@/components/locale-document-language\"")
    expect(source).not.toContain("<LocaleDocumentLanguage />")
  })
})
