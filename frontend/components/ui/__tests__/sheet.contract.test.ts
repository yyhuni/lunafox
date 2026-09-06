import { describe, expect, it } from "vitest"
import { readdirSync, readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/sheet.tsx"), "utf8")

function collectSourceFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const entryPath = path.join(directory, entry.name)
    if (entry.isDirectory()) return collectSourceFiles(entryPath)
    return entry.isFile() && /\.[jt]sx$/.test(entry.name) ? [entryPath] : []
  })
}

describe("sheet contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared edge overlay style tokens", () => {
    expect(source).toContain("from \"@/lib/ui/overlay-styles\"")
    expect(source).toContain("edgeOverlayBackdropClassName")
    expect(source).toContain("edgeOverlayPanelBaseClassName")
  })

  it("uses Base UI dialog primitives for the sheet backend", () => {
    expect(source).toContain('from "@base-ui/react/dialog"')
    expect(source).not.toContain("@radix-ui/react-dialog")
    expect(source).toContain("SheetPrimitive.Backdrop")
    expect(source).toContain("SheetPrimitive.Popup")
  })

  it("stages controlled first mounts through the shared edge-panel lifecycle", () => {
    expect(source).toContain("useStagedEdgePanelOpen")
    expect(source).toContain("const stagedOpen = useStagedEdgePanelOpen(open)")
    expect(source).toContain("open={stagedOpen}")
  })

  it("uses Base UI render directly instead of Radix-style asChild compatibility", () => {
    expect(source).not.toContain("asChild")
    expect(source).not.toContain("render={asChild ? child : render}")
  })

  it("maps the retained escape callback to Base UI open-change cancellation", () => {
    expect(source).toContain("onEscapeKeyDown?: (event: KeyboardEvent) => void")
    expect(source).toContain('eventDetails.reason === "escape-key"')
    expect(source).toContain("eventDetails.cancel()")
  })

  it("keeps every edge panel on the shared directional motion contract", () => {
    expect(source).toContain("className={cn(edgeOverlayBackdropClassName, className)}")
    expect(source).toContain("<SheetOverlay />")
    expect(source).not.toContain("overlayClassName")
    expect(source).not.toContain("overlayBackdropBlur")
    expect(source).not.toContain("backdropBlur")
    expect(source).not.toContain("overlayBackdropBaseClassName")
    expect(source).not.toContain("sideMotion")
    expect(source).toContain('"data-starting-style:translate-x-full data-ending-style:translate-x-full"')
    expect(source).toContain('"data-starting-style:-translate-x-full data-ending-style:-translate-x-full"')
  })

  it("uses the shared panel title role for sheet titles", () => {
    expect(source).toContain("textRole.panelTitle")
    expect(source).not.toContain("className={cn(textRole.sectionTitle, className)}")
  })

  it("renders an accessible shared close control by default", () => {
    expect(source).toContain("showCloseButton = true")
    expect(source).toContain("showCloseButton?: boolean")
    expect(source).toContain("{showCloseButton && (")
    expect(source).toContain("<SheetPrimitive.Close")
    expect(source).toContain('aria-label={tActions("close")}')
    expect(source).toContain("<semanticIcons.action.cancel />")
    expect(source).toContain('import { semanticIcons } from "@/components/icons"')
  })

  it("allows close-control opt-out only for reviewed production callers", () => {
    const componentsRoot = path.resolve(process.cwd(), "components")
    const optOutFiles = collectSourceFiles(componentsRoot)
      .filter((file) => readFileSync(file, "utf8").includes("showCloseButton={false}"))
      .map((file) => path.relative(process.cwd(), file))
      .sort()

    expect(optOutFiles).toEqual([
      "components/notifications/notification-drawer-sections.tsx",
      "components/scan/scheduled/create-scheduled-scan-dialog.tsx",
      "components/shared/detail-drawer/detail-drawer.tsx",
      "components/shared/form-drawer/form-drawer.tsx",
    ])
  })
})
