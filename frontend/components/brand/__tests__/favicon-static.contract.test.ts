import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const layoutSource = readFileSync(path.resolve(process.cwd(), "app/layout.tsx"), "utf8")
const appFaviconPath = path.resolve(process.cwd(), "app/favicon.ico")
const appIconPath = path.resolve(process.cwd(), "app/icon.png")
const appAppleIconPath = path.resolve(process.cwd(), "app/apple-icon.png")
const brandReadme = readFileSync(path.resolve(process.cwd(), "components/brand/README.md"), "utf8")

describe("favicon-static contract", () => {
  it("uses app-level metadata files instead of route-local icon wiring", () => {
    expect(layoutSource).not.toContain("icons: {")
    expect(existsSync(appFaviconPath)).toBe(true)
    expect(existsSync(appIconPath)).toBe(true)
    expect(existsSync(appAppleIconPath)).toBe(true)
    expect(layoutSource).not.toContain('from "@/components/brand/favicon-sync"')
    expect(layoutSource).not.toContain("<FaviconSync />")
  })

  it("documents the app-level favicon ownership rule", () => {
    expect(brandReadme).toContain("Browser favicon MUST use Next.js app-level metadata files in `frontend/app`.")
    expect(brandReadme).not.toContain("runtime favicon update path")
    expect(brandReadme).not.toContain("runtime PNG favicon path")
  })
})
