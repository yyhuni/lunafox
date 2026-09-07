import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/layout.tsx"), "utf8")

describe("layout contract", () => {
  it("moves durable app providers and boot ownership to the root layout", () => {
    expect(source).toContain("export default async function RootLayout")
    expect(source).toContain("export async function generateMetadata")
    expect(source).toContain("from \"react\"")
    expect(source).toContain('from "next-intl/server"')
    expect(source).toContain('from "next-intl"')
    expect(source).toContain('from "@/i18n/locale"')
    expect(source).toContain("<html")
    expect(source).toContain("<body")
    expect(source).toContain("<ColorThemeInit")
    expect(source).toContain("<BootLayerController />")
    expect(source).toContain("<ThemeProvider>")
    expect(source).toContain("<NextIntlClientProvider")
    expect(source).toContain("<AuthLayout>")
    expect(source).toContain("<Toaster />")
    expect(source).toContain("resolveRequestLocale()")
    expect(source).toContain("loadLocaleMessages(locale)")
    expect(source).not.toContain("[locale]")
  })
})
