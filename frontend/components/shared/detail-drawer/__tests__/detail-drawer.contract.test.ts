import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/detail-drawer/detail-drawer.tsx"), "utf8")
const indexSource = readFileSync(path.resolve(process.cwd(), "components/shared/detail-drawer/index.ts"), "utf8")

describe("detail-drawer shared contract", () => {
  it("exports a shared right-side detail drawer shell", () => {
    expect(source).toContain("export function DetailDrawer")
    expect(source).toContain("SheetContent")
    expect(source).toContain('side="right"')
    expect(source).toContain("detailDrawerContentClassName")
    expect(source).not.toContain("detailDrawerBackdropClassName")
    expect(source).not.toContain("overlayClassName")
    expect(source).not.toContain("sideMotion")
    expect(source).toContain("SheetHeader")
    expect(source).toContain("SheetTitle")
    expect(source).toContain("SheetClose")
    expect(source).toContain("SheetDescription")
    expect(source).toContain("titleMeta")
    expect(source).toContain("headerMeta")
    expect(source).toContain("sidecar")
    expect(source).toContain("sidecarClassName")
    expect(source).not.toContain("motion =")
  })

  it("keeps the detail drawer shell on the shared card surface", () => {
    expect(source).toContain('"relative flex h-full min-h-0 flex-col bg-card"')
    expect(source).toContain('"flex h-full min-h-0 flex-1 flex-col bg-card"')
    expect(source).not.toContain('flex-col bg-background")')
  })

  it("inherits the shared sheet panel title role instead of page-local title sizing", () => {
    expect(source).toContain('const titleText = typeof title === "string" ? title : undefined')
    expect(source).toContain('className={cn("min-w-0 truncate", titleClassName)}')
    expect(source).toContain("title={titleText}")
    expect(source).toContain("EdgePanelHeader")
    expect(source).toContain('variant="detail"')
    expect(source).not.toContain("text-xl")
  })

  it("delegates detail header geometry and metadata slots to the shared detail variant", () => {
    expect(source).toContain('titleMeta={titleMeta}')
    expect(source).toContain('headerMeta={headerMeta}')
    expect(source).toContain('actions={(')
    expect(source).toContain('className="shrink-0"')
  })

  it("keeps the drawer close affordance on the shared quiet icon button", () => {
    expect(source).toContain('showCloseButton={false}')
    expect(source).toContain('render={<Button type="button" variant="ghost" size="icon-sm" aria-label={tActions("close")} className="shrink-0"/>}')
    expect(source).toContain("<semanticIcons.action.cancel />")
    expect(source).toContain('import { semanticIcons } from "@/components/icons"')
    expect(source).not.toContain("focus:ring-")
  })

  it("supports a left sidecar column for secondary workflows", () => {
    expect(source).toContain("absolute inset-0")
    expect(source).toContain("lg:right-full")
    expect(source).toContain("lg:left-auto")
    expect(source).toContain("lg:max-w-lg")
    expect(source).toContain("lg:border-r")
    expect(source).toContain("lg:animate-in")
    expect(source).toContain("lg:slide-in-from-right-2")
  })

  it("lets sidecar workflows intercept Escape before closing the detail drawer", () => {
    expect(source).toContain("onSidecarClose")
    expect(source).toContain("onEscapeKeyDown")
    expect(source).toContain("event.preventDefault()")
  })

  it("exports shared content tab primitives for detail drawers", () => {
    expect(source).toContain("export function DetailDrawerTabs")
    expect(source).toContain("DetailDrawerTabsList")
    expect(source).toContain("DetailDrawerTabsTrigger")
    expect(source).toContain("DetailDrawerTabsContent")
    expect(source).toContain('variant = "content"')
    expect(source).toContain("variant={variant}")
    expect(source).toContain("TabsContent")
  })

  it("keeps drawer tabs as quiet secondary navigation instead of primary page chrome", () => {
    expect(source).toContain("border-border/70\"")
    expect(source).toContain("data-active:border-foreground")
    expect(source).toContain("data-active:text-foreground")
    expect(source).toContain("focus-visible:ring-2 focus-visible:ring-ring/40")
    expect(source).not.toContain("data-active:border-primary")
  })

  it("keeps the public entrypoint small and explicit", () => {
    expect(indexSource).toContain("DetailDrawer")
    expect(indexSource).toContain("DetailDrawerTabs")
    expect(indexSource).toContain("DetailDrawerTabsList")
    expect(indexSource).toContain("DetailDrawerTabsTrigger")
    expect(indexSource).toContain("DetailDrawerTabsContent")
  })
})
