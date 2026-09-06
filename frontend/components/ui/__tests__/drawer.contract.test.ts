import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/drawer.tsx"), "utf8")
const globals = readFileSync(path.resolve(process.cwd(), "app/globals.css"), "utf8")
const overlaySource = readFileSync(path.resolve(process.cwd(), "lib/ui/overlay-styles.ts"), "utf8")

describe("drawer contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared edge overlay style tokens", () => {
    expect(source).toContain("from \"@/lib/ui/overlay-styles\"")
    expect(source).toContain("drawerOverlayBackdropClassName")
    expect(source).toContain("drawerPopupClassName")
  })

  it("uses the shadcn Base Drawer overlay and transition hooks", () => {
    expect(source).toContain("className={cn(drawerOverlayBackdropClassName, className)}")
    expect(source).toContain("data-snap-points={hasSnapPoints ? \"\" : undefined}")
    expect(source).toContain("data-[swipe-direction=right]:[--closed-transform:translate3d(calc(100%+var(--drawer-inset)+2px),0,0)]")
    expect(overlaySource).toContain("data-starting-style:transform-(--closed-transform)")
  })

  it("stages controlled first mounts through the shared edge-panel lifecycle", () => {
    expect(source).toContain("useStagedEdgePanelOpen")
    expect(source).toContain("const stagedOpen = useStagedEdgePanelOpen(open)")
    expect(source).toContain("open={stagedOpen}")
  })

  it("coordinates compositor-friendly panel and backdrop motion", () => {
    expect(overlaySource).toContain("transition-transform duration-[var(--motion-duration-shell)] ease-[var(--motion-ease-standard)]")
    expect(overlaySource).toContain("transition-opacity duration-[var(--motion-duration-shell)] ease-[var(--motion-ease-standard)]")
    expect(overlaySource).not.toContain("transition-[transform,height,opacity,filter]")
    expect(overlaySource).not.toContain("duration-450")
    expect(overlaySource).toContain("`${drawerOverlayBackdropBaseClassName} supports-backdrop-filter:backdrop-blur-xs`")
    expect(overlaySource).toContain('export const drawerPanelMotionClassName =\n  "transform-gpu will-change-transform"')
    expect(overlaySource).not.toContain("workbenchDrawerPanelMotionClassName")
    expect(overlaySource).toContain("data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)]")
    expect(overlaySource).toContain("data-swiping:duration-0")
  })

  it("uses the shared card surface for ordinary drawer shells", () => {
    expect(overlaySource).toMatch(/drawerPopupClassName\s*=\s*\n\s*"[^"\n]*bg-card[^"\n]*text-card-foreground/)
  })

  it("does not expose caller-level drawer backdrop variants", () => {
    expect(source).not.toContain("overlayClassName")
    expect(source).not.toContain("overlayBackdropBlur")
    expect(source).not.toContain("backdropBlur")
    expect(source).not.toContain("drawerOverlayBackdropBaseClassName")
  })

  it("keeps the global body positioned for Base UI drawer overlays", () => {
    expect(globals).toMatch(/body\s*\{[\s\S]*position:\s*relative;/)
  })

  it("uses Base UI drawer semantics instead of Vaul", () => {
    expect(source).toContain("@base-ui/react/drawer")
    expect(source).toContain("useDrawer must be used within a Drawer")
    expect(source).toContain("modal = true")
    expect(source).toContain("showSwipeHandle")
    expect(source).toContain("swipeDirection")
    expect(source).toContain("data-swipe-axis={swipeAxis}")
    expect(source).toContain("DrawerSwipeHandle")
    expect(source).toContain("data-[swipe-direction=right]")
    expect(source).not.toContain("from \"vaul\"")
    expect(source).not.toContain("data-[vaul-drawer-direction")
  })

  it("uses the shared panel title role for drawer titles", () => {
    expect(source).toContain("textRole.panelTitle")
    expect(source).not.toContain("\"text-foreground font-semibold\"")
  })
})
