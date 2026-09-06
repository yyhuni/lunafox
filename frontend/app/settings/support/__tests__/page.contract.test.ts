import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const filePath = path.resolve(process.cwd(), "app/settings/support/page.tsx")
const workspacePath = path.resolve(process.cwd(), "app/settings/support/support-workspace.tsx")
const contentPath = path.resolve(process.cwd(), "components/settings/support/support-page-content.tsx")
const loadingStatePath = path.resolve(process.cwd(), "components/settings/support/support-page-loading-state.tsx")
const layoutPath = path.resolve(process.cwd(), "components/settings/support/support-page-layout.tsx")

describe("support page contract", () => {
  it("keeps the route shell server-rendered", () => {
    expect(existsSync(filePath)).toBe(true)

    const source = readFileSync(filePath, "utf8")
    expect(source).toContain("export default function SupportPage")
    expect(source).toContain("<SupportWorkspace />")
    expect(source).toContain("SupportWorkspace")
    expect(source).not.toContain("\"use client\"")
    expect(source).not.toContain("next/dynamic")
  })

  it("routes hidden readiness through the shared route-boundary helper", () => {
    expect(existsSync(workspacePath)).toBe(true)

    const workspaceSource = readFileSync(workspacePath, "utf8")
    expect(workspaceSource).toContain("support-page-loading-state")
    expect(workspaceSource).toContain("HiddenReadinessRouteBoundary")
    expect(workspaceSource).toContain("loading: () => null")
    expect(workspaceSource).toContain("skeleton={<SupportPageLoadingState />}")
    expect(workspaceSource).toContain('layer="workspace"')
    expect(workspaceSource).toContain('intent="data"')
    expect(workspaceSource).not.toContain("support-page-skeleton")
    expect(workspaceSource).not.toContain("SupportPageSkeleton")
    expect(workspaceSource).toContain("SupportPageContent")
    expect(workspaceSource).toContain("onReady={onReady}")
    expect(workspaceSource).toContain("deferInitialSkeleton={deferInitialSkeleton}")
    expect(workspaceSource).not.toContain("const [isReady, setIsReady] = React.useState(false)")
  })

  it("lets the route handoff own initial loading visibility", () => {
    const contentSource = readFileSync(contentPath, "utf8")

    expect(contentSource).toContain("onReady?: () => void")
    expect(contentSource).toContain("deferInitialSkeleton?: boolean")
    expect(contentSource).toContain("onReady?.()")
  })

  it("derives loading and content route geometry from the same support layout owner", () => {
    const contentSource = readFileSync(contentPath, "utf8")
    const loadingStateSource = readFileSync(loadingStatePath, "utf8")
    const layoutSource = readFileSync(layoutPath, "utf8")

    expect(layoutSource).toContain("SUPPORT_PAGE_ROUTE_SURFACE_CLASS")
    expect(layoutSource).toContain("min-h-[max(70vh,calc(100vh-4rem))]")
    expect(layoutSource).toContain("SupportPageLayout")
    expect(contentSource).toContain("SupportPageLayout")
    expect(contentSource).toContain('data-testid="support-value-first"')
    expect(loadingStateSource).toContain("export function SupportPageLoadingState")
    expect(loadingStateSource).toContain("SupportPageLayout")
    expect(loadingStateSource).toContain("ActionSkeleton")
    expect(contentSource).not.toContain('data-loading-slot="surface"')
    expect(loadingStateSource).not.toContain('data-loading-slot="surface"')
    expect(contentSource).not.toContain('className="h-[480px] w-[320px]')

    for (const slot of ["support-page-header", "support-page-value-band", "support-page-actions"]) {
      expect(layoutSource).toContain(`data-loading-slot=\"${slot}\"`)
    }

    expect(layoutSource).toContain('"data-loading-slot": "support-page-tier-options"')
    expect(contentSource).toContain("SUPPORT_PAGE_TIER_OPTIONS_LOADING_SLOT")
    expect(loadingStateSource).toContain("SUPPORT_PAGE_TIER_OPTIONS_LOADING_SLOT")
  })

  it("keeps loading ownership inside the support workspace", () => {
    const workspaceSource = readFileSync(workspacePath, "utf8")
    expect(workspaceSource).toContain("SupportPageLoadingState")
    expect(workspaceSource).not.toContain("RouteSegmentLoadingOwner")
    expect(workspaceSource).not.toContain("RouteFallback")
  })
})
