import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/auth/auth-layout.tsx"), "utf8")
const protectedShellSource = readFileSync(path.resolve(process.cwd(), "components/auth/protected-app-shell.ts"), "utf8")
const protectedAuthLayoutPath = path.resolve(process.cwd(), "components/auth/protected-auth-layout.tsx")
const protectedAuthLayoutSource = existsSync(protectedAuthLayoutPath)
  ? readFileSync(protectedAuthLayoutPath, "utf8")
  : ""
const globals = readFileSync(path.resolve(process.cwd(), "app/globals.css"), "utf8")

describe("auth-layout contract", () => {
  it("uses shared auth runtime helpers instead of duplicating public route logic", () => {
    expect(source).toContain("export function AuthLayout")
    expect(source).toContain("from \"@/lib/auth-runtime.mjs\"")
    expect(source).not.toContain("const PUBLIC_ROUTES")
    expect(source).toContain("buildLoginRedirectPath")
    expect(source).toContain("buildReturnTo")
  })

  it("does not rely on important utility overrides for the root layout shell", () => {
    expect(source).not.toContain("!min-h-0")
  })

  it("keeps the protected shell on semantic background surfaces", () => {
    expect(protectedAuthLayoutSource).toContain("bg-background flex flex-1 min-h-0")
  })

  it("owns one session-renewal coordinator at the authenticated shell boundary", () => {
    expect(protectedAuthLayoutSource).toContain('from "@/hooks/use-session-renewal"')
    expect(protectedAuthLayoutSource).toContain("useSessionRenewal()")
  })

  it("keeps auth gate in AuthLayout — unauthenticated never mounts protected shell", () => {
    expect(source).toContain("if (!hydrated || isLoading || !auth?.authenticated)")
    expect(source).toContain("function AuthBootHandoffBlocker()")
    expect(source).toContain('data-boot-handoff-pending="true"')
    expect(source).toContain('data-testid="auth-boot-handoff-blocker"')
    expect(source).not.toContain('owner="auth-layout-pending"')
    expect(source).not.toContain('intent="auth"')
    expect(source).not.toContain('import { AppWarmupLoader } from "@/components/shared/loading/app-warmup-loader"')
    expect(source).not.toContain("const shouldRenderApp = isLoading || !!auth?.authenticated")
    expect(source).not.toContain("active={showLoading}")
  })

  it("only renders ProtectedAuthLayout after authentication is confirmed", () => {
    expect(source).toContain("<ProtectedAuthLayout>")
    expect(source).not.toContain("hydrated={hydrated}")
    expect(source).not.toContain("isLoading={isLoading}")
    expect(source).not.toContain("authenticated={auth?.authenticated}")
  })

  it("ProtectedAuthLayout no longer handles auth state branching", () => {
    expect(protectedAuthLayoutSource).not.toContain("hydrated: boolean")
    expect(protectedAuthLayoutSource).not.toContain("isLoading: boolean")
    expect(protectedAuthLayoutSource).not.toContain("authenticated?: boolean")
    expect(protectedAuthLayoutSource).not.toContain("auth-layout-warmup")
    expect(protectedAuthLayoutSource).not.toContain("auth-layout-redirect-warmup")
    expect(protectedAuthLayoutSource).not.toContain("AppShellWarmup")
  })

  it("keeps the root boot layer visible while protected shell and route children suspend", () => {
    expect(source).toContain("fallback={<AuthBootHandoffBlocker />}")
    expect(source).not.toContain("AppShellWarmup")
    expect(protectedAuthLayoutSource).not.toContain('import { AppShellWarmup }')
    expect(protectedAuthLayoutSource).not.toContain('import { AuthWarmupShell }')
    expect(protectedAuthLayoutSource).not.toContain("<AuthWarmupShell")
    expect(protectedAuthLayoutSource).toContain("function RouteBootHandoffBlocker()")
    expect(protectedAuthLayoutSource).toContain("fallback={<RouteBootHandoffBlocker />}")
  })

  it("reveals protected route content through the shared subtle content reveal owner", () => {
    expect(protectedAuthLayoutSource).toContain('import { ContentReveal } from "@/components/shared/loading/content-reveal"')
    expect(protectedAuthLayoutSource).toMatch(
      /<ContentReveal\s+owner="auth-layout-route-content"/
    )
    expect(protectedAuthLayoutSource).toContain('className="flex min-h-0 flex-1 flex-col"')
  })

  it("no longer delays shell warmup for auth — auth gate is in AuthLayout", () => {
    expect(protectedAuthLayoutSource).not.toContain("AUTH_LAYOUT_WARMUP_DELAY_MS")
    expect(source).toContain("AuthBootHandoffBlocker")
  })

  it("uses a desktop left-right shell with the sidebar before the top bar", () => {
    expect(protectedAuthLayoutSource).toContain('import { AppSidebar } from "@/components/app-sidebar"')
    expect(protectedAuthLayoutSource).toContain('import { UnifiedHeader } from "@/components/unified-header"')
    expect(protectedAuthLayoutSource).toContain('protectedAppShellContentFrameClassName,')
    expect(protectedAuthLayoutSource).toContain('protectedAppShellScrollAreaClassName,')
    expect(protectedAuthLayoutSource).toContain('protectedAppShellStyle,')
    expect(protectedAuthLayoutSource).toContain('className="flex h-svh min-h-0 w-full"')
    expect(protectedAuthLayoutSource).toContain('className="flex flex-1 flex-col min-h-0"')
    expect(protectedAuthLayoutSource).toContain("className={protectedAppShellScrollAreaClassName}")
    expect(protectedAuthLayoutSource).toContain("className={protectedAppShellContentFrameClassName}")
    expect(protectedAuthLayoutSource).toContain("style={protectedAppShellStyle}")
    expect(protectedAuthLayoutSource.indexOf("<AppSidebar />")).toBeLessThan(protectedAuthLayoutSource.indexOf("<UnifiedHeader />"))
    expect(protectedShellSource).not.toContain('"--sidebar-width"')
    expect(protectedShellSource).toContain('"--sidebar-top": "0px"')
    expect(protectedShellSource).toContain('"--sidebar-height": "100svh"')
    expect(protectedShellSource).toContain('"--header-height": "calc(var(--spacing) * 10)"')
    expect(protectedShellSource).toContain("export const protectedAppShellScrollAreaClassName")
    expect(protectedShellSource).toContain("overflow-x-hidden overflow-y-auto [scrollbar-gutter:stable_both-edges]")
    expect(protectedShellSource).toContain("export const protectedAppShellContentFrameClassName")
    expect(protectedShellSource).toContain("mx-auto")
    expect(protectedShellSource).toContain("max-w-none")
    expect(protectedShellSource).not.toContain("max-w-screen-2xl")
    expect(protectedShellSource).not.toContain("getProtectedAppShellContentFrameClassName")
    expect(protectedShellSource).not.toContain("isProtectedOverviewPathname")
    expect(protectedAuthLayoutSource).not.toContain('from "next/navigation"')
  })

  it("does not store route-specific table sizing variables in the app shell", () => {
    expect(source).not.toContain("--vuln-toolbar-h")
    expect(source).not.toContain("--vuln-table-head-h")
    expect(source).not.toContain("--vuln-row-h")
    expect(protectedAuthLayoutSource).not.toContain("--vuln-toolbar-h")
    expect(protectedAuthLayoutSource).not.toContain("--vuln-table-head-h")
    expect(protectedAuthLayoutSource).not.toContain("--vuln-row-h")
  })

  it("does not use empty dynamic fallbacks for layout-critical shell chrome", () => {
    expect(source).not.toContain("const AppSidebar = dynamic")
    expect(source).not.toContain("const UnifiedHeader = dynamic")
    expect(source).not.toContain("loading: () => null")
    expect(protectedAuthLayoutSource).not.toContain("const AppSidebar = dynamic")
    expect(protectedAuthLayoutSource).not.toContain("const UnifiedHeader = dynamic")
    expect(protectedAuthLayoutSource).not.toContain("loading: () => null")
  })

  it("does not draw target or scan child fallback scenes before their workspace owner", () => {
    expect(protectedAuthLayoutSource).not.toContain("function DetailChildRouteLoadingState")
    expect(protectedAuthLayoutSource).not.toContain("DetailPageShellSkeleton")
    expect(protectedAuthLayoutSource).not.toContain("WebSitesViewRouteFallback")
    expect(protectedAuthLayoutSource).not.toContain("SubdomainsDetailViewRouteFallback")
    expect(protectedAuthLayoutSource).not.toContain("IPAddressesViewRouteFallback")
    expect(protectedAuthLayoutSource).not.toContain("EndpointsDetailViewRouteFallback")
    expect(protectedAuthLayoutSource).not.toContain("DirectoriesViewRouteFallback")
    expect(protectedAuthLayoutSource).not.toContain("ScreenshotsGalleryRouteFallback")
    expect(protectedAuthLayoutSource).not.toContain("VulnerabilitiesDetailViewRouteFallback")
    expect(protectedAuthLayoutSource).not.toContain("TargetSettingsRouteFallback")
  })

  it("does not substitute a visible route fallback for workflow suspense", () => {
    expect(protectedAuthLayoutSource).toContain("fallback={<RouteBootHandoffBlocker />}")
    expect(protectedAuthLayoutSource).not.toContain("ScanWorkflowPageLoadingState")
    expect(protectedAuthLayoutSource).not.toContain("scan-workflow-route-suspense")
  })

  it("keeps protected shell modules out of the public auth wrapper bundle", () => {
    expect(source).toContain("React.lazy(() => import(\"@/components/auth/protected-auth-layout\")")
    expect(source).toContain("<ProtectedAuthLayout")
    expect(source).not.toContain('from "@/components/app-sidebar"')
    expect(source).not.toContain('from "@/components/unified-header"')
    expect(source).not.toContain('from "@/components/scan/history/scan-overview-sections"')
    expect(protectedAuthLayoutSource).not.toContain('from "@/components/scan/history/scan-overview-loading-state"')
    expect(protectedAuthLayoutSource).not.toContain('from "@/components/scan/history/scan-overview-sections"')
    expect(source).not.toContain('from "@/components/target/target-overview-sections"')
    expect(source).not.toContain('from "@/components/organization/organization-detail-view-sections"')
  })

  it("does not stack a whole-shell reveal on top of route and workspace handoff", () => {
    expect(protectedAuthLayoutSource).not.toContain("animate-app-fade-in")
    expect(globals).not.toContain(".animate-app-fade-in")
    expect(globals).not.toContain("@keyframes app-fade-in")
    expect(globals).toContain("loading-content-reveal")
    expect(globals).not.toContain("filter: blur(1px)")
    expect(globals).not.toContain("translateY(2px)")
  })

  it("keeps the login shader fallback theme-owned without glow fields", () => {
    const start = globals.indexOf(".auth-shader-fallback")
    const block = globals.slice(start, globals.indexOf("}", start))

    expect(start).toBeGreaterThan(-1)
    expect(block).toContain("var(--background)")
    expect(block).toContain("var(--primary)")
    expect(globals).not.toContain(".auth-shader-fallback__dots")
    expect(globals).not.toContain(".auth-shader-fallback__dot")
    expect(globals).not.toContain("@keyframes auth-shader-fallback-dot")
    expect(block).not.toContain("radial-gradient")
  })
})
