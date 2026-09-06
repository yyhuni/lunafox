import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/providers/ui-i18n-provider.tsx"), "utf8")

describe("ui-i18n-provider contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function UiI18nProvider")
    expect(source).toContain("from \"next-intl\"")
  })
})
