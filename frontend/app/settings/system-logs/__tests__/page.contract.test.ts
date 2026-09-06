import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/settings/system-logs/page.tsx"), "utf8")
const workspaceSource = readFileSync(path.resolve(process.cwd(), "app/settings/system-logs/system-logs-workspace.tsx"), "utf8")
const viewSource = readFileSync(path.resolve(process.cwd(), "components/settings/system-logs/system-logs-view.tsx"), "utf8")
const loadingStateSource = readFileSync(path.resolve(process.cwd(), "components/settings/system-logs/system-logs-loading-state.tsx"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "components/settings/system-logs/system-logs-layout.ts"), "utf8")

describe("page contract", () => {
  it("keeps the route shell server-rendered", () => {
    expect(source).toContain("export default function SystemLogsPage")
    expect(source).toContain("SystemLogsWorkspace")
    expect(source).toContain("<SystemLogsWorkspace />")
    expect(source).not.toContain("\"use client\"")
    expect(source).not.toContain("from \"next/dynamic\"")
    expect(source).not.toContain("force-dynamic")
  })

  it("uses the system logs view-owned loading state for route loading", () => {
    expect(workspaceSource).toContain("SystemLogsLoadingState")
    expect(workspaceSource).toContain('import dynamic from "next/dynamic"')
    expect(workspaceSource).toContain('import type { SystemLogsViewProps } from "@/components/settings/system-logs/system-logs-view"')
    expect(workspaceSource).toContain("const SystemLogsView = dynamic<SystemLogsViewProps>(")
    expect(workspaceSource).toContain("loading: () => null")
    expect(workspaceSource).toContain("HiddenReadinessRouteBoundary")
    expect(workspaceSource).toContain("owner=\"system-logs-page-route\"")
    expect(workspaceSource).toContain('layer="workspace"')
    expect(workspaceSource).toContain('intent="data"')
    expect(workspaceSource).toContain('const pageTitle = t("title")')
    expect(workspaceSource).toContain('const pageDescription = t("description")')
    expect(workspaceSource).toContain("<SystemLogsLoadingState")
    expect(workspaceSource).toContain("pageTitle={pageTitle}")
    expect(workspaceSource).toContain("pageDescription={pageDescription}")
    expect(workspaceSource).toContain("<SystemLogsView")
    expect(workspaceSource).not.toContain("SystemLogsPageSkeleton")
    expect(workspaceSource).not.toContain("system-logs-page-skeleton")
    expect(workspaceSource).toContain("SYSTEM_LOGS_WORKSPACE_HANDOFF_CLASS")
    expect(workspaceSource).toContain("SYSTEM_LOGS_WORKSPACE_STATE_CLASS")
    expect(workspaceSource).toContain("className={SYSTEM_LOGS_WORKSPACE_HANDOFF_CLASS}")
    expect(workspaceSource).toContain("skeletonClassName={SYSTEM_LOGS_WORKSPACE_STATE_CLASS}")
    expect(workspaceSource).toContain("contentClassName={SYSTEM_LOGS_WORKSPACE_STATE_CLASS}")
    expect(workspaceSource).toContain("onReady={onReady}")
    expect(workspaceSource).toContain("deferInitialSkeleton={deferInitialSkeleton}")
    expect(workspaceSource).not.toContain("const [isReady, setIsReady] = React.useState(false)")
    expect(workspaceSource).not.toContain('import { SystemLogsView } from "@/components/settings/system-logs/system-logs-view"')
    expect(workspaceSource).not.toContain("lazyPage(")
    expect(workspaceSource).toContain("system-logs-loading-state")
    expect(workspaceSource).not.toContain("RouteSegmentLoadingOwner")
    expect(workspaceSource).not.toContain("RouteFallback")
  })

  it("lets the route handoff own initial log loading visibility", () => {
    expect(viewSource).toContain("onReady?: () => void")
    expect(viewSource).toContain("deferInitialSkeleton?: boolean")
    expect(viewSource).toContain("onReady?.()")
    expect(viewSource).toContain('data-loading-hidden-readiness={deferInitialSkeleton ? "true" : undefined}')
    expect(viewSource).not.toMatch(/if \(isInitialLoading && deferInitialSkeleton\)\s+return null/)
  })

  it("keeps the loading header on the same PageHeader rhythm", () => {
    expect(loadingStateSource).toContain("export function SystemLogsLoadingState")
    expect(loadingStateSource).toContain("PageHeader")
    expect(loadingStateSource).toContain("title={pageTitle}")
    expect(loadingStateSource).toContain("description={pageDescription}")
    expect(loadingStateSource).toContain("opacity-0")
    expect(loadingStateSource).toContain('code="LOG-01"')
    expect(loadingStateSource).not.toContain('className="space-y-3 px-4 lg:px-6"')
  })

  it("keeps the loading status bar on the same responsive rhythm", () => {
    expect(loadingStateSource).toContain("SYSTEM_LOGS_DEFAULT_LINES")
    expect(loadingStateSource).toContain("toolbar.serverSource")
    expect(loadingStateSource).toContain("toolbar.autoRefresh")
    expect(loadingStateSource).toContain("Separator")
    expect(loadingStateSource).toContain("Switch")
  })

  it("keeps terminal loading and content in the same viewport-bound workbench", () => {
    expect(layoutSource).toContain('SYSTEM_LOGS_WORKSPACE_HANDOFF_CLASS =\n  "flex h-full min-h-0 flex-1 flex-col"')
    expect(layoutSource).toContain('SYSTEM_LOGS_WORKSPACE_STATE_CLASS = "flex h-full min-h-0 flex-1 flex-col"')
  })
})
