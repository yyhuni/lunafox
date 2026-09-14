import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const typographyPath = path.resolve(process.cwd(), "lib/typography.ts")

describe("typography foundation contract", () => {
  it("exposes a shared typography role module", () => {
    expect(existsSync(typographyPath)).toBe(true)
  })

  it("defines the production text roles used by foundation components", () => {
    if (!existsSync(typographyPath)) {
      expect.fail("lib/typography.ts is missing")
    }

    const source = readFileSync(typographyPath, "utf8")
    const requiredRoles = [
      "pageTitle",
      "pageTitleDisplay",
      "pageDescription",
      "panelTitle",
      "sectionTitle",
      "compactSectionTitle",
      "compactPrimary",
      "compactCaption",
      "body",
      "bodyLarge",
      "bodyStrong",
      "bodySubtle",
      "helperText",
      "metadataLabel",
      "metadataValue",
      "navLabel",
      "tableHeader",
      "tableCellPrimary",
      "tableCellSecondary",
      "badge",
      "badgeSubtle",
      "tab",
      "code",
      "metricValueDisplay",
    ]

    expect(source).toContain("export const textRole")
    for (const role of requiredRoles) {
      expect(source).toContain(`${role}:`)
    }
  })

  it("keeps the production-readable scale centered on 20/16/13/12/11 roles", () => {
    if (!existsSync(typographyPath)) {
      expect.fail("lib/typography.ts is missing")
    }

    const source = readFileSync(typographyPath, "utf8")

    expect(source).toContain('pageTitle: "text-xl')
    expect(source).toContain('pageTitleDisplay:')
    expect(source).toContain('panelTitle: "text-base')
    expect(source).toContain('sectionTitle: "text-[13px] font-semibold')
    expect(source).toContain('metricValueDisplay:')
    expect(source).toContain('metricValueDisplay: "[font-family:var(--font-display)] text-xl font-medium text-foreground leading-none tracking-tight tabular-nums data-[featured=true]:text-3xl data-[featured=true]:font-semibold"')
    expect(source).toContain('bodyLarge: "text-base')
    expect(source).toContain('body: "text-sm')
    expect(source).toContain('helperText: "text-xs')
    expect(source).toContain('compactSectionTitle: "text-xs font-semibold text-foreground leading-4 tracking-normal"')
    expect(source).toContain('compactPrimary: "text-xs font-medium text-foreground leading-4 tracking-normal"')
    expect(source).toContain('compactCaption: "text-[11px] font-normal text-muted-foreground leading-4 tracking-normal"')
    expect(source).toContain('tableHeader: "text-[12px]')
    expect(source).toContain('tab: "text-[13px]')
    expect(source).toContain('badge: "text-[11px]')
  })

  it("keeps Chinese-readable roles free of default uppercase and excessive tracking", () => {
    if (!existsSync(typographyPath)) {
      expect.fail("lib/typography.ts is missing")
    }

    const source = readFileSync(typographyPath, "utf8")
    for (const role of ["tableHeader", "badge", "badgeSubtle", "tab", "sectionTitle"]) {
      const match = source.match(new RegExp(`${role}:\\s*"([^"]+)"`))
      expect(match?.[1] ?? "").not.toContain("uppercase")
      expect(match?.[1] ?? "").not.toContain("tracking-[0.12em]")
      expect(match?.[1] ?? "").not.toContain("tracking-[0.14em]")
      expect(match?.[1] ?? "").not.toContain("tracking-[0.2em]")
    }
  })

  it("makes dense table body text line-height explicit at 20px", () => {
    if (!existsSync(typographyPath)) {
      expect.fail("lib/typography.ts is missing")
    }

    const source = readFileSync(typographyPath, "utf8")
    expect(source).toContain('tableCellPrimary: "text-[13px] font-medium text-foreground leading-[18px] tracking-normal"')
    expect(source).toContain('tableCellSecondary: "text-xs font-normal text-muted-foreground leading-4 tracking-normal"')
  })

  it("makes compact table-header and badge line-height explicit at 16px", () => {
    if (!existsSync(typographyPath)) {
      expect.fail("lib/typography.ts is missing")
    }

    const source = readFileSync(typographyPath, "utf8")
    expect(source).toContain('tableHeader: "text-[12px] font-medium text-muted-foreground leading-4 tracking-normal"')
    expect(source).toContain('badge: "text-[11px] font-medium leading-4 tracking-normal normal-case"')
    expect(source).toContain('badgeSubtle: "text-[11px] font-medium text-muted-foreground leading-4 tracking-normal normal-case"')
  })

  it("makes tab, helper, and pagination-adjacent metadata line-heights explicit", () => {
    if (!existsSync(typographyPath)) {
      expect.fail("lib/typography.ts is missing")
    }

    const source = readFileSync(typographyPath, "utf8")
    expect(source).toContain('helperText: "text-xs font-normal text-muted-foreground leading-4 tracking-normal"')
    expect(source).toContain('metadataLabel: "text-xs font-normal text-muted-foreground leading-4 tracking-normal"')
    expect(source).toContain('metadataValue: "text-xs font-normal text-foreground leading-4 tracking-normal"')
    expect(source).toContain('metadataValueStrong: "text-xs font-medium text-foreground leading-4 tracking-normal"')
    expect(source).toContain('tab: "text-[13px] font-medium leading-[18px] tracking-normal normal-case"')
  })
})
