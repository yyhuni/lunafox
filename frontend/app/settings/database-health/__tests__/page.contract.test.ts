import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/settings/database-health/page.tsx"), "utf8")
const workspaceSource = readFileSync(path.resolve(process.cwd(), "app/settings/database-health/database-health-workspace.tsx"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "components/settings/database-health/database-health-layout.ts"), "utf8")

describe("page contract", () => {
  it("keeps the route shell server-rendered", () => {
    expect(source).toContain("export default function DatabaseHealthPage")
    expect(source).toContain("DatabaseHealthWorkspace")
    expect(source).toContain("<DatabaseHealthWorkspace />")
    expect(source).not.toContain("\"use client\"")
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toContain("getTranslations")
    expect(source).not.toContain("force-dynamic")
  })

  it("preserves current route-boundary markers in the workspace", () => {
    expect(workspaceSource).toContain("dynamic<DatabaseHealthViewProps>")
    expect(workspaceSource).toContain("HiddenReadinessRouteBoundary")
    expect(workspaceSource).toContain("DatabaseHealthLoadingState")
    expect(workspaceSource).toContain('from "@/components/settings/database-health/database-health-loading-state"')
    expect(workspaceSource).toContain('const pageTitle = tPage("title")')
    expect(workspaceSource).toContain('const pageDescription = tPage("description")')
    expect(workspaceSource).toContain("<DatabaseHealthLoadingState")
    expect(workspaceSource).toContain("pageTitle={pageTitle}")
    expect(workspaceSource).toContain("pageDescription={pageDescription}")
    expect(workspaceSource).toContain("<DatabaseHealthView")
    expect(workspaceSource).toContain("loading: () => null")
    expect(workspaceSource).toContain('owner="database-health-page-route"')
    expect(workspaceSource).toContain('layer="workspace"')
    expect(workspaceSource).toContain('intent="data"')
    expect(workspaceSource).toContain("onReady={onReady}")
    expect(workspaceSource).toContain("deferInitialSkeleton={deferInitialSkeleton}")
    expect(workspaceSource).not.toContain("DatabaseHealthPageSkeleton")
    expect(workspaceSource).not.toContain("DatabaseHealthRouteSkeleton")
    expect(workspaceSource).not.toContain("database-health-page-skeleton")
    expect(workspaceSource).not.toContain("DatabaseHealthRouteFallback")
    expect(workspaceSource).not.toContain("const [isReady, setIsReady] = React.useState(false)")
    expect(workspaceSource).not.toContain("lazyPage(")
    expect(workspaceSource).not.toContain("from \"@/components/common/lazy-page\"")
  })

  it("keeps loading ownership inside the database health workspace", () => {
    expect(workspaceSource).toContain("<DatabaseHealthLoadingState")
    expect(workspaceSource).not.toContain('owner="database-health-page"')
    expect(workspaceSource).not.toContain("RouteSegmentLoadingOwner")
    expect(workspaceSource).not.toContain("RouteFallback")
  })

  it("keeps the first-screen database workspace viewport-stable across loading and content", () => {
    expect(workspaceSource).toContain("DATABASE_HEALTH_WORKSPACE_HANDOFF_CLASS")
    expect(workspaceSource).toContain("DATABASE_HEALTH_WORKSPACE_STATE_CLASS")
    expect(workspaceSource).toContain("className={DATABASE_HEALTH_WORKSPACE_HANDOFF_CLASS}")
    expect(workspaceSource).toContain("skeletonClassName={DATABASE_HEALTH_WORKSPACE_STATE_CLASS}")
    expect(workspaceSource).toContain("contentClassName={DATABASE_HEALTH_WORKSPACE_STATE_CLASS}")
    expect(layoutSource).toContain('DATABASE_HEALTH_WORKSPACE_HANDOFF_CLASS =\n  "flex h-full min-h-0 flex-1 flex-col"')
    expect(layoutSource).toContain('DATABASE_HEALTH_WORKSPACE_STATE_CLASS = "flex h-full min-h-0 flex-1 flex-col"')
    expect(layoutSource).toContain('DATABASE_HEALTH_CONTENT_SHELL_CLASS =\n  "min-h-0 flex-1 space-y-4 overflow-y-auto px-4 [scrollbar-gutter:stable] lg:px-6"')
  })
})
