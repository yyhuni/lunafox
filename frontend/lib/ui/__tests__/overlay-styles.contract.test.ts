import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/ui/overlay-styles.ts"), "utf8")

describe("overlay-styles contract", () => {
  it("exports shared overlay style tokens", () => {
    expect(source).toContain("export const floatingContentMotionClassName")
    expect(source).toContain("export const floatingSurfaceClassName")
    expect(source).toContain("export const overlayBackdropClassName")
    expect(source).toContain("export const edgeOverlayBackdropClassName")
    expect(source).toContain("export const drawerOverlayBackdropClassName")
    expect(source).toContain("export const drawerPopupClassName")
    expect(source).toContain("export const drawerPanelMotionClassName")
    expect(source).toContain("export const centeredOverlayPanelClassName")
    expect(source).toContain("export const formDrawerContentClassName")
    expect(source).toContain("export const scanWorkbenchDrawerContentClassName")
    expect(source).toContain("export const editorDialogPanelClassName")
    expect(source).toContain("export const edgeOverlayPanelBaseClassName")
    expect(source).toContain("export const detailDrawerContentClassName")
    expect(source).toContain("export const workbenchLogDrawerContentClassName")
    expect(source).toContain("export const compactFeedbackDrawerContentClassName")
    expect(source).toContain("export const tooltipSurfaceClassName")
    expect(source).toContain("export const tooltipArrowClassName")
  })

  it("owns distinct header and sidebar offsets for shell-bound overlays", () => {
    expect(source).toContain("export const shellOverlaySideOffsets")
    expect(source).toContain("header: 8")
    expect(source).toContain("sidebarDesktop: 16")
    expect(source).not.toContain("sidebarAccountDesktop")
    expect(source).not.toContain("sidebarNavigationDesktop")
  })

  it("does not export caller-level backdrop bypass tokens", () => {
    expect(source).not.toMatch(/export const overlayBackdropBaseClassName/)
    expect(source).not.toMatch(/export const drawerOverlayBackdropBaseClassName/)
    expect(source).not.toContain("export const detailDrawerBackdropClassName")
    expect(source).not.toContain("export const formDrawerBackdropClassName")
    expect(source).not.toContain("backdrop-blur-0")
  })

  it("uses semantic overlay surfaces instead of page-background shells", () => {
    expect(source).toContain('bg-popover p-4 text-popover-foreground')
    expect(source).toContain("bg-card")
    expect(source).toContain("text-card-foreground")
    expect(source).not.toContain("rounded-lg border bg-background p-6")
  })

  it("keeps edge-panel motion compositor-friendly and synchronized", () => {
    expect(source).toContain("bg-(--overlay-backdrop-background)")
    expect(source).toContain("supports-backdrop-filter:backdrop-blur-xs")
    expect(source).toContain("transition-opacity duration-[var(--motion-duration-shell)] ease-[var(--motion-ease-standard)]")
    expect(source).toContain("transition-transform duration-[var(--motion-duration-shell)] ease-[var(--motion-ease-standard)]")
    expect(source).not.toContain("transition-[transform,height,opacity,filter]")
    expect(source).not.toContain("overlayBackdropClassName =\n  `${overlayBackdropBaseClassName} backdrop-blur-sm`")
  })

  it("keeps shared floating motion wired to Base UI popup state attributes", () => {
    expect(source).toContain("data-[open]:animate-in")
    expect(source).toContain("data-[closed]:animate-out")
    expect(source).toContain("data-[starting-style]:zoom-in-95")
    expect(source).toContain("data-[ending-style]:zoom-out-95")
  })

  it("keeps drawer panel motion separate from the shared edge-panel backdrop", () => {
    expect(source).toContain("drawerPanelMotionClassName")
    expect(source).toContain("transform-gpu will-change-transform")
  })

  it("keeps wide edge-panel layouts separate from their shared motion owner", () => {
    expect(source).toContain("drawerPanelMotionClassName")
    expect(source).toContain("formDrawerContentClassName =")
    expect(source).not.toContain("data-[open]:slide-in-from-right-8")
    expect(source).not.toContain("data-[closed]:slide-out-to-right-8")
    expect(source).toContain("workbenchLogDrawerContentClassName")
    expect(source).toContain("sm:max-w-4xl")
    expect(source).toContain("scanWorkbenchDrawerContentClassName")
    expect(source).toContain("xl:max-w-4xl")
    expect(source).toContain("compactFeedbackDrawerContentClassName")
    expect(source).toContain("sm:max-w-[420px]")
  })

  it("prevents drawer panels from growing past their viewport-owned width", () => {
    expect(source).toContain("fixed z-50 flex min-w-0 max-w-full flex-col")
    expect(source).toContain("flex min-w-0 w-full max-w-full flex-col gap-0 overflow-hidden")
    expect(source).toContain("min-w-0 w-full max-w-full p-0 sm:max-w-5xl")
    expect(source).toContain("min-w-0 w-full max-w-full gap-0 p-0 sm:max-w-4xl")
    expect(source).toContain("flex min-w-0 w-full max-w-full flex-col gap-0 p-0 sm:max-w-[420px]")
  })

  it("keeps scan workbench drawers composed from the shared workbench motion token", () => {
    expect(source).toMatch(/scanWorkbenchDrawerContentClassName =\s*`\$\{drawerPanelMotionClassName\}[\s\S]*xl:max-w-4xl`/)
  })

  it("does not reintroduce Radix-style overlay state selectors for Base UI motion", () => {
    expect(source).not.toContain("data-[state=open]")
    expect(source).not.toContain("data-[state=closed]")
  })
})
