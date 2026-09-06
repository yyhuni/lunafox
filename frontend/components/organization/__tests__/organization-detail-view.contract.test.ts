import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/organization-detail-view.tsx"), "utf8")

describe("organization-detail-view contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function OrganizationDetailView")
    expect(source).toContain("export function OrganizationDetailDrawerView")
    expect(source).toContain("from \"./organization-detail-view-sections\"")
    expect(source).toContain("OrganizationDetailViewWorkbench")
  })

  it("keeps empty and error states from flashing before the initial handoff completes", () => {
    expect(source).toContain("ContentHandoff")
    expect(source).toContain("const isInitialLoading = state.isInitialContentLoading && !state.error")
    expect(source).toContain("if (!isInitialLoading && state.error)")
    expect(source).toContain("if (!isInitialLoading && !state.organization)")
    expect(source).toContain('owner="organization-detail-view-content"')
    expect(source).toContain("skeleton={<OrganizationDetailViewLoadingState state={state} />}")
    expect(source).toContain("prepareContentBeforeHandoff")
    expect(source).toContain("<OrganizationDetailViewWorkbench state={state} />")
    expect(source).toContain('from "@/components/shared/feedback/app-error-state"')
    expect(source).toContain("<AppErrorState")
    expect(source).not.toContain("OrganizationDetailViewErrorState")
    expect(source).not.toContain("ContentReveal")
  })

  it("renders drawer sidecar actions as a sibling column to the detail drawer", () => {
    expect(source).toContain("sidecar={sidecar}")
    expect(source).toContain("handleSidecarClose")
    expect(source).toContain("onSidecarClose={handleSidecarClose}")
    expect(source).toContain("OrganizationDetailActionPanel")
    expect(source).toContain('owner="organization-detail-drawer-content"')
    expect(source).toContain("<OrganizationDetailViewLoadingState")
    expect(source).toContain("state={state}")
    expect(source).toContain('summarySurface="drawer"')
    expect(source).toContain('previewName={typeof description === "string" ? description : undefined}')
    expect(source).toContain("previewDescription={previewDescription}")
  })

  it("mounts the detail loading owner with the drawer instead of staging a delayed body", () => {
    expect(source).toContain('<div className="min-h-0 flex-1 overflow-y-auto py-5">')
    expect(source).not.toContain("DETAIL_DRAWER_CONTENT_DEFER_MS")
    expect(source).not.toContain("shouldRenderBody")
    expect(source).not.toContain("prefers-reduced-motion: reduce")
    expect(source).not.toContain('motion="workbench"')
  })
})
