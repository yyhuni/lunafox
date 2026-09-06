import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/settings/api-keys/page.tsx"), "utf8")
const workspaceSource = readFileSync(path.resolve(process.cwd(), "app/settings/api-keys/api-keys-settings-workspace.tsx"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "app/settings/api-keys/api-keys-settings-layout.ts"), "utf8")
const contentSource = readFileSync(path.resolve(process.cwd(), "app/settings/api-keys/content.tsx"), "utf8")
const loadingStateSource = readFileSync(
  path.resolve(process.cwd(), "components/settings/api-keys/api-keys-settings-loading-state.tsx"),
  "utf8"
)

describe("page contract", () => {
  it("keeps the route shell server-rendered", () => {
    expect(source).toContain("export default function ApiKeysSettingsPage")
    expect(source).toContain("ApiKeysSettingsWorkspace")
    expect(source).toContain("<ApiKeysSettingsWorkspace />")
    expect(source).not.toContain("\"use client\"")
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toContain("getTranslations")
    expect(source).not.toContain("force-dynamic")
  })

  it("routes hidden readiness through the shared route-boundary helper", () => {
    expect(workspaceSource).toContain("dynamic<ApiKeysSettingsPageProps>")
    expect(workspaceSource).toContain("HiddenReadinessRouteBoundary")
    expect(workspaceSource).toContain("ApiKeysSettingsLoadingState")
    expect(workspaceSource).toContain('from "@/components/settings/api-keys/api-keys-settings-loading-state"')
    expect(workspaceSource).toContain("loading: () => null")
    expect(workspaceSource).toContain('owner="api-keys-page-route"')
    expect(workspaceSource).toContain('layer="workspace"')
    expect(workspaceSource).toContain('intent="data"')
    expect(workspaceSource).toContain('const pageTitle = t("title")')
    expect(workspaceSource).toContain('const pageDescription = t("description")')
    expect(workspaceSource).toContain("<ApiKeysSettingsLoadingState")
    expect(workspaceSource).toContain("pageTitle={pageTitle}")
    expect(workspaceSource).toContain("pageDescription={pageDescription}")
    expect(workspaceSource).toContain("<ApiKeysSettingsPageContent")
    expect(workspaceSource).toContain("onReady={onReady}")
    expect(workspaceSource).toContain("deferInitialSkeleton={deferInitialSkeleton}")
    expect(workspaceSource).not.toContain("import { ApiKeysSettingsSkeleton }")
    expect(workspaceSource).not.toContain("ApiKeysSettingsRouteSkeleton")
    expect(workspaceSource).not.toContain("api-keys-settings-skeleton")
    expect(workspaceSource).not.toContain("ApiKeysSettingsRouteFallback")
    expect(workspaceSource).not.toContain("const [isReady, setIsReady] = React.useState(false)")
    expect(workspaceSource).not.toContain("lazyPage(")
  })

  it("keeps loading ownership inside the API key workspace", () => {
    expect(workspaceSource).toContain("<ApiKeysSettingsLoadingState")
    expect(workspaceSource).not.toContain('owner="api-keys-page"')
    expect(workspaceSource).not.toContain("RouteSegmentLoadingOwner")
    expect(workspaceSource).not.toContain("RouteFallback")
  })

  it("keeps the first-screen workbench viewport-stable across the route handoff", () => {
    expect(workspaceSource).toContain("API_KEYS_WORKSPACE_HANDOFF_CLASS")
    expect(workspaceSource).toContain("API_KEYS_WORKSPACE_STATE_CLASS")
    expect(workspaceSource).toContain("className={API_KEYS_WORKSPACE_HANDOFF_CLASS}")
    expect(workspaceSource).toContain("skeletonClassName={API_KEYS_WORKSPACE_STATE_CLASS}")
    expect(workspaceSource).toContain("contentClassName={API_KEYS_WORKSPACE_STATE_CLASS}")
    expect(layoutSource).toContain('API_KEYS_WORKSPACE_HANDOFF_CLASS =\n  "flex h-full min-h-0 flex-1 flex-col"')
    expect(layoutSource).toContain('API_KEYS_WORKSPACE_STATE_CLASS = "flex h-full min-h-0 flex-1 flex-col"')
  })

  it("pairs the route-critical API key regions across loading and content", () => {
    for (const slot of [
      "api-keys-header",
      "api-keys-provider-list",
      "api-keys-provider-detail",
      "api-keys-notice",
    ]) {
      expect(loadingStateSource).toContain(`getLoadingStructureSlotAttributes(\"${slot}\")`)
      expect(contentSource).toContain(`getLoadingStructureSlotAttributes(\"${slot}\")`)
    }
  })
})
