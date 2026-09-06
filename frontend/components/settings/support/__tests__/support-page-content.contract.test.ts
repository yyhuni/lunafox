import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const filePath = path.resolve(process.cwd(), "components/settings/support/support-page-content.tsx")
const loadingStatePath = path.resolve(process.cwd(), "components/settings/support/support-page-loading-state.tsx")
const layoutPath = path.resolve(process.cwd(), "components/settings/support/support-page-layout.tsx")
const alipayQrPath = path.resolve(process.cwd(), "public/images/support/alipay-qr.jpg")
const wechatQrPath = path.resolve(process.cwd(), "public/images/support/wechat-qr.jpg")
const contactQrPath = path.resolve(process.cwd(), "public/images/support/contact-qr.jpg")

describe("support-page-content contract", () => {
  it("creates the dedicated support page content component", () => {
    expect(existsSync(filePath)).toBe(true)

    const source = readFileSync(filePath, "utf8")
    expect(source).toContain("export default function SupportPageContent")
    expect(existsSync(loadingStatePath)).toBe(true)
    expect(source).toContain('data-testid="support-value-first"')
    expect(source).not.toContain("ScratchOverlay")
    expect(source).not.toContain("lockedTitle")
    expect(source).toContain('data-testid={`support-tier-${tier.id}`}')
    expect(source).toContain('useTranslations("settings.support")')
    expect(source).toContain('src={`/images/support/${activeMethod}-qr.jpg`}')
    expect(source).toContain("Dialog")
    expect(source).toContain('data-testid="support-dialog-amount"')
    expect(source).toContain('data-testid="support-dialog-title"')
    expect(source).toContain('data-testid={`payment-method-${method}`}')
    expect(source).toContain('const SUPPORT_METHODS = ["wechat", "alipay", "contact"] as const')
    expect(source).toContain('t("dialog.title")')
    expect(source).toContain('activeMethod !== "contact"')
    expect(source).not.toContain("transition-all")
    expect(source).not.toContain("group-hover:translate-x")
    expect(source).not.toContain("group-hover:-translate-y")
    expect(source).not.toContain("animate-pulse")
    expect(source).not.toContain('t("usage.title")')
    expect(source).not.toContain('t("contact.title")')
    expect(source).not.toContain("/images/support/alipay-qr.svg")
    expect(source).not.toContain("/images/support/wechat-qr.svg")
    expect(source).not.toContain('ctx.font = "bold 20px sans-serif"')
    expect(source).not.toContain('ctx.font = "bold 16px sans-serif"')
    expect(source).not.toContain('ctx.font = "12px sans-serif"')
    expect(existsSync(alipayQrPath)).toBe(true)
    expect(existsSync(wechatQrPath)).toBe(true)
    expect(existsSync(contactQrPath)).toBe(true)
  })

  it("keeps the value-first content and loading state on the shared layout contract", () => {
    const source = readFileSync(filePath, "utf8")
    const loadingSource = readFileSync(loadingStatePath, "utf8")
    const layoutSource = readFileSync(layoutPath, "utf8")
    const sharedLayoutConstants = [
      "SUPPORT_PAGE_ROUTE_SURFACE_CLASS",
      "SUPPORT_PAGE_TIER_OPTIONS_LOADING_SLOT",
      "SUPPORT_TIER_CARD_CLASS",
    ]
    const sharedLayoutOwners = [
      "SupportPageLayout",
      "SupportPageContentStack",
      "SupportPageHeader",
      "SupportPageValueBand",
      "SupportPageValueBandList",
      "SupportPageValueBandItem",
      "SupportPageActions",
      "SupportPageFooter",
    ]

    for (const constant of sharedLayoutConstants) {
      expect(layoutSource).toContain(`export const ${constant}`)
    }

    for (const layoutOwner of sharedLayoutOwners) {
      expect(layoutSource).toContain(`export function ${layoutOwner}`)
      expect(source).toContain(layoutOwner)
      expect(loadingSource).toContain(layoutOwner)
    }

    for (const slot of ["support-page-header", "support-page-value-band", "support-page-actions"]) {
      expect(layoutSource).toContain(`data-loading-slot=\"${slot}\"`)
    }

    expect(layoutSource).toContain('"data-loading-slot": "support-page-tier-options"')
    expect(source).toContain("SUPPORT_PAGE_TIER_OPTIONS_LOADING_SLOT")
    expect(loadingSource).toContain("SUPPORT_PAGE_TIER_OPTIONS_LOADING_SLOT")

    expect(loadingSource).toContain("SupportPageLoadingState")
    expect(loadingSource).toContain("getLoadingOwnerAttributes")
    expect(loadingSource).toContain("ActionSkeleton")
    expect(loadingSource).toContain("Skeleton")
    expect(source).not.toContain("support-page-skeleton")
    expect(source).not.toContain("SUPPORT_LOCKED_CARD_BODY_CLASS")
    expect(source).not.toContain("min-h-[160px]")
    expect(layoutSource).toContain("items-center justify-start overflow-x-clip p-4 py-8 md:p-8")
    expect(layoutSource).toContain("footer?: React.ReactNode")
    expect(layoutSource).toContain("flex-col justify-start md:min-h-0 md:flex-1 md:justify-center")
    expect(layoutSource).toContain("mt-10 w-full max-w-5xl md:mt-0")
    expect(source).toContain("footer={")
    expect(loadingSource).toContain("footer={")
    expect(layoutSource).not.toContain('data-loading-slot="surface"')
    expect(source).not.toContain('className="text-2xl font-bold tracking-tight text-foreground"')
    expect(source).not.toContain('className="px-2 text-sm leading-relaxed text-muted-foreground"')
    expect(source).not.toContain('className="mt-4 text-[11px] font-semibold uppercase tracking-widest text-primary/80"')
  })

  it("keeps the component-owned loading state independent from support interaction dependencies", () => {
    const loadingSource = readFileSync(loadingStatePath, "utf8")

    expect(loadingSource).not.toContain("framer-motion")
    expect(loadingSource).not.toContain("canvas-confetti")
    expect(loadingSource).not.toContain("next/image")
  })
})
