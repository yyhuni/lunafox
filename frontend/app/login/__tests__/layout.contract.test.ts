import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/login/layout.tsx"), "utf8")

describe("layout contract", () => {
  it("keeps login metadata at the canonical route without locale params", () => {
    expect(source).toContain("export default function LoginLayout")
    expect(source).toContain("export async function generateMetadata")
    expect(source).toContain("resolveRequestLocale()")
    expect(source).toContain("getTranslations({ locale, namespace: \"auth\" })")
    expect(source).not.toContain("params: Promise<{ locale:")
  })
})
