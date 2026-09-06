import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/settings/notifications/page.tsx"), "utf8")
const workspaceSource = readFileSync(path.resolve(process.cwd(), "app/settings/notifications/notification-settings-workspace.tsx"), "utf8")

describe("page contract", () => {
  it("keeps the route shell server-rendered", () => {
    expect(source).toContain("export default function NotificationSettingsPage")
    expect(source).toContain("NotificationSettingsWorkspace")
    expect(source).toContain("<NotificationSettingsWorkspace />")
    expect(source).not.toContain("\"use client\"")
    expect(source).not.toContain("from \"next/dynamic\"")
    expect(source).not.toContain("force-dynamic")
  })

  it("routes hidden readiness through the shared route-boundary helper", () => {
    expect(workspaceSource).toContain("HiddenReadinessRouteBoundary")
    expect(workspaceSource).toContain("NotificationSettingsPageLoadingState")
    expect(workspaceSource).toContain("useNotificationDestinations")
    expect(workspaceSource).toContain('import NotificationSettingsPageContent from "@/components/settings/notifications/notification-settings-page-content"')
    expect(workspaceSource).toContain("owner=\"notification-settings-page-route\"")
    expect(workspaceSource).toContain('layer="workspace"')
    expect(workspaceSource).toContain('intent="data"')
    expect(workspaceSource).toContain('const pageTitle = t("pageTitle")')
    expect(workspaceSource).toContain('const pageDescription = t("pageDesc")')
    expect(workspaceSource).toContain("<NotificationSettingsPageLoadingState")
    expect(workspaceSource).toContain("pageTitle={pageTitle}")
    expect(workspaceSource).toContain("pageDescription={pageDescription}")
    expect(workspaceSource).toContain("destinations={destinationsQuery.data?.results}")
    expect(workspaceSource).toContain("supportedKindCount={destinationsQuery.data?.supportedKinds.length}")
    expect(workspaceSource).toContain("<NotificationSettingsPageContent")
    expect(workspaceSource).not.toContain("notification-settings-page-skeleton")
    expect(workspaceSource).not.toContain("NotificationSettingsPageSkeleton")
    expect(workspaceSource).toContain("onReady={onReady}")
    expect(workspaceSource).toContain("deferInitialSkeleton={deferInitialSkeleton}")
    expect(workspaceSource).not.toContain("const [isReady, setIsReady] = React.useState(false)")
    expect(workspaceSource).not.toContain("ContentHandoff")
    expect(workspaceSource).not.toContain("next/dynamic")
    expect(workspaceSource).not.toContain("loading: () => null")
  })

  it("keeps localized loading ownership inside the destination workspace", () => {
    expect(workspaceSource).toContain("notification-settings-loading-state")
    expect(workspaceSource).not.toContain("NotificationSettingsRouteLoadingState")
    expect(workspaceSource).not.toContain("RouteSegmentLoadingOwner")
    expect(workspaceSource).not.toContain("RouteFallback")
  })
})
