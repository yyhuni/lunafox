import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const workspace = readFileSync(path.resolve(process.cwd(), "components/settings/login-visual/login-visual-settings-workspace.tsx"), "utf8")
const loginPage = readFileSync(path.resolve(process.cwd(), "components/auth/visual-split-login.tsx"), "utf8")
const route = readFileSync(path.resolve(process.cwd(), "app/settings/login-visual/page.tsx"), "utf8")
const sidebar = readFileSync(path.resolve(process.cwd(), "components/app-sidebar.tsx"), "utf8")

describe("login visual settings composition", () => {
  it("exposes the System Settings route from the shared sidebar", () => {
    expect(route).toContain('import { LoginVisualSettingsWorkspace }')
    expect(route).toContain("return <LoginVisualSettingsWorkspace />")
    expect(sidebar).toContain("title: t('loginVisual')")
    expect(sidebar).toContain('url: "/settings/login-visual/"')
  })

  it("keeps replacement, draft publication, and recovery in the workspace", () => {
    expect(workspace).toContain("<VisualSplitLogin")
    expect(workspace).toContain("visual: previewVisualFor(asset)")
    expect(workspace).toContain("useLoginPreviewScale")
    expect(workspace).toContain('className="flex h-full min-h-0 flex-1 flex-col gap-4 py-4 md:gap-6 md:py-6"')
    expect(workspace).toContain('className="flex min-h-0 flex-1 flex-col overflow-y-auto px-4 [scrollbar-gutter:stable] lg:px-6"')
    expect(workspace).toContain('className="flex w-full flex-1 flex-col justify-center"')
    expect(workspace).toContain('role="button"')
    expect(workspace).toContain('accept="image/jpeg,image/png,image/webp,video/mp4,video/webm"')
    expect(workspace).toContain("URL.createObjectURL(file)")
    expect(workspace).toContain("URL.revokeObjectURL")
    expect(workspace).toContain("useLoginVisualPreview")
    expect(workspace).toContain("acceptedPreview.source")
    expect(workspace).toContain("upload.mutate(file")
    expect(workspace).toContain("onError: () =>")
    expect(workspace).toContain("publish.mutate()")
    expect(workspace).toContain("restore.mutate()")
  })

  it("keeps the first-screen preview behind the shared loading handoff", () => {
    expect(workspace).toContain("HiddenReadinessRouteBoundary")
    expect(workspace).toContain('owner="login-visual-settings-page-route"')
    expect(workspace).toContain('getLoadingStructureSlotAttributes("login-visual-header")')
    expect(workspace).toContain('getLoadingStructureSlotAttributes("login-visual-preview")')
    expect(workspace).toContain('getLoadingStructureSlotAttributes("login-visual-publication-controls")')
    expect(workspace).toContain("onReady={onReady}")
    expect(workspace).toContain("deferInitialSkeleton={deferInitialSkeleton}")
  })

  it("reuses the full desktop login page and only layers the replacement control onto its visual panel", () => {
    expect(loginPage).toContain("preview?: VisualSplitLoginPreview")
    expect(loginPage).toContain("preview?.visualControl")
    expect(loginPage).toContain('preview ? "h-full min-h-0" : "min-h-svh"')
    expect(loginPage).toContain('id={preview ? "login-visual-preview-email" : "email"}')
  })
})
