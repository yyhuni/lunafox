import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/scan/scheduled/page.tsx"), "utf8")
const workspaceSource = readFileSync(path.resolve(process.cwd(), "app/scan/scheduled/scheduled-scan-workspace.tsx"), "utf8")

describe("page contract", () => {
  it("keeps the route shell server-rendered", () => {
    expect(source).toContain("export default function ScheduledScanPage")
    expect(source).toContain("ScheduledScanWorkspace")
    expect(source).not.toContain("\"use client\"")
    expect(source).not.toContain("from \"next/dynamic\"")
  })

  it("keeps the route dynamic fallback invisible so the page owns the only visible skeleton", () => {
    expect(workspaceSource).toContain("from \"@/components/shared/loading/hidden-readiness-route-boundary\"")
    expect(workspaceSource).toContain("from \"@/components/scan/scheduled/scheduled-scan-page-sections\"")
    expect(workspaceSource).toContain("import type { ScheduledScanPageProps }")
    expect(workspaceSource).toContain("dynamic<ScheduledScanPageProps>")
    expect(workspaceSource).toContain("loading: () => null")
    expect(workspaceSource).not.toContain("loading: () => <ScheduledScanPageLoadingState />")
    expect(workspaceSource).not.toContain("scheduled-scan-page-skeleton")
    expect(workspaceSource).not.toContain("from \"@/components/common/lazy-page\"")
  })

  it("routes scheduled-scan hidden readiness through the shared route-boundary helper", () => {
    expect(workspaceSource).toContain("<HiddenReadinessRouteBoundary")
    expect(workspaceSource).toContain("owner=\"scheduled-scan-page-route\"")
    expect(workspaceSource).toContain('layer="workspace"')
    expect(workspaceSource).toContain('intent="data"')
    expect(workspaceSource).toContain("skeleton={<ScheduledScanPageLoadingState />}")
    expect(workspaceSource).toContain("onReady={onReady}")
    expect(workspaceSource).toContain("deferInitialSkeleton={deferInitialSkeleton}")
    expect(workspaceSource).not.toContain("const [isReady, setIsReady] = React.useState(false)")
  })
})
