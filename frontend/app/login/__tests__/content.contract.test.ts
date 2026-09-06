import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/login/content.tsx"), "utf8")
const prefetchSource = readFileSync(
  path.resolve(process.cwd(), "hooks/use-overview-prefetch.ts"),
  "utf8"
)

describe("content contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default function LoginPage")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("keeps post-login progress on the login surface instead of a fullscreen auth warmup", () => {
    expect(source).not.toContain("AUTH_WARMUP_DELAY_MS = 120")
    expect(source).not.toContain("import { AppWarmupLoader }")
    expect(source).not.toContain("showWarmup && !showExitOverlay")
    expect(source).toContain("showLoginSurface ? (")
    expect(source).toContain("shouldKeepLoginSurfaceForHydration")
    expect(source).toContain("authLoading || auth?.authenticated !== true")
    expect(source).toContain("loginVisualReady")
    expect(source).toContain("data-boot-handoff-pending")
    expect(source).toContain("className=\"flex w-full justify-center\"")
    expect(source).not.toContain("document.readyState")
    expect(source).not.toContain("lunafox:route-prefetch-done")
  })

  it("does not render legacy trace animations", () => {
    expect(source).not.toContain("trace trace-")
    expect(source).not.toContain("trace-glow")
    expect(source).not.toContain("animation: traceFlow 3s linear infinite;")
    expect(source).not.toContain("animation: traceFlowV 3s linear infinite;")
  })

  it("renders the self-owned animated ascii background layer", () => {
    expect(source).not.toContain("import { LoginAsciiBackground }")
    expect(source).not.toContain("<LoginAsciiBackground")
    expect(source).not.toContain("login-ascii-background")
  })

  it("renders the visual split login block while keeping auth flow ownership in the route", () => {
    expect(source).toContain("import { VisualSplitLogin }")
    expect(source).toContain("<VisualSplitLogin")
    expect(source).toContain("onLogin={handleLogin}")
    expect(source).toContain("onVisualReady={handleLoginVisualReady}")
    expect(source).toContain("useTranslations(\"auth.visualLogin\")")
    expect(source).toContain("translations={{")
  })

  it("resolves post-login navigation from a canonical returnTo instead of rebuilding locale-prefixed paths", () => {
    expect(source).toContain("useSearchParams")
    expect(source).toContain("resolveSafeReturnTo")
    expect(source).toContain("DEFAULT_AUTH_RETURN_TO")
    expect(source).toContain("const defaultReturnTo = DEFAULT_AUTH_RETURN_TO")
    expect(source).toContain("const nextRoute = safeReturnTo ?? defaultReturnTo")
    expect(source).toContain("replaceWithRouteProgress(router, nextRoute)")
    expect(source).toContain("router.prefetch(nextRoute)")
    expect(source).not.toContain("const withLocale = React.useCallback")
    expect(source).not.toContain("replaceWithRouteProgress(router, withLocale(\"/overview/\"))")
  })

  it("keeps post-login data warmup out of the login entry bundle", () => {
    expect(source).not.toContain("import { vulnerabilityKeys } from \"@/hooks/use-vulnerabilities\"")
    expect(source).not.toContain("import { getAssetStatistics, getStatisticsHistory } from \"@/services/overview.service\"")
    expect(source).not.toContain("import { getScans } from \"@/services/scan.service\"")
    expect(source).not.toContain("import { VulnerabilityService } from \"@/services/vulnerability.service\"")
    expect(source).toContain("import { usePrefetchOverviewData } from \"@/hooks/use-overview-prefetch\"")
    expect(source).toContain("const prefetchOverviewData = usePrefetchOverviewData()")
    expect(source).not.toContain("import(\"@/services/overview.service\")")
    expect(source).not.toContain("import(\"@/services/scan.service\")")
    expect(source).not.toContain("import(\"@/services/vulnerability.service\")")
    expect(prefetchSource).toContain("import(\"@/services/overview.service\")")
    expect(prefetchSource).toContain("import(\"@/services/scan.service\")")
    expect(prefetchSource).toContain("import(\"@/services/vulnerability.service\")")
    expect(prefetchSource).toContain("import(\"@/hooks/use-overview\")")
    expect(prefetchSource).toContain("import(\"@/hooks/use-scans/keys\")")
    expect(prefetchSource).toContain("import(\"@/hooks/use-vulnerabilities/keys\")")
  })

  it("lets the login surface inherit the runtime color theme", () => {
    expect(source).not.toContain("AUTH_LOGIN_THEME_ID")
    expect(source).not.toContain("data-theme={")
    expect(source).toContain('data-auth-theme="public"')
    expect(source).not.toContain("useColorTheme")
    expect(source).not.toContain("useTheme")
  })

  it("does not render the legacy login grid layer", () => {
    expect(source).not.toContain("circuit-grid")
  })
})
