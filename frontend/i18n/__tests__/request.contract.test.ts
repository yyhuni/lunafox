import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "i18n/request.ts"), "utf8")

describe("request contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain('from "next-intl/server"')
    expect(source).toContain("resolveRequestLocale")
    expect(source).not.toContain("requestLocale")
  })
})
