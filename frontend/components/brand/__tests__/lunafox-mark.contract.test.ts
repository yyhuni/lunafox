import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/brand/lunafox-mark.tsx"), "utf8")
const globals = readFileSync(path.resolve(process.cwd(), "app/globals.css"), "utf8")
const themeBase = readFileSync(path.resolve(process.cwd(), "styles/themes/base.css"), "utf8")

describe("lunafox-mark contract", () => {
  it("owns the shared brand mark and favicon svg helper", () => {
    expect(source).toContain("export function LunaFoxMark")
    expect(source).toContain("export function getLunaFoxFaviconSvg")
    expect(source).toContain("export async function getLunaFoxFaviconPngDataUrl")
    expect(source).toContain("export async function getLunaFoxBrandedFaviconPngDataUrl")
    expect(source).toContain("FAVICON_MASK_DATA_URI")
    expect(source).toContain("background: string")
    expect(source).toContain("foreground: string")
    expect(source).toContain('data-slot="lunafox-mark"')
    expect(source).toContain('mask="url(#fox-mask)"')
    expect(source).toContain('href="${FAVICON_MASK_DATA_URI}"')
  })

  it("renders the mark through shared mask styling and theme tokens", () => {
    expect(globals).toContain(".lunafox-mark")
    expect(globals).toContain('mask-image: url("/images/brand/fox-mark.png")')
    expect(globals).toContain("background-color: var(--brand-mark-foreground);")
    expect(themeBase).toContain("--brand-mark-foreground")
  })
})
