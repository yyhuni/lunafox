import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/dialog.tsx"), "utf8")
const overlaySource = readFileSync(path.resolve(process.cwd(), "lib/ui/overlay-styles.ts"), "utf8")

describe("dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared modal overlay style tokens", () => {
    expect(source).toContain("from \"@/lib/ui/overlay-styles\"")
    expect(source).toContain("overlayBackdropClassName")
    expect(source).toContain("centeredOverlayPanelClassName")
  })

  it("uses Base UI as the dialog primitive backend", () => {
    expect(source).toContain('from "@base-ui/react/dialog"')
    expect(source).not.toContain("@radix-ui/react-dialog")
    expect(source).toContain("DialogPrimitive.Popup")
    expect(source).toContain("DialogPrimitive.Backdrop")
  })

  it("uses Base UI render directly instead of Radix-style asChild compatibility", () => {
    expect(source).not.toContain("asChild")
    expect(source).not.toContain("render={asChild ? child : render}")
  })

  it("exports shared dialog width tiers for route-level forms", () => {
    expect(overlaySource).toContain("scrollableFormDialogContentClassName")
    expect(overlaySource).toContain("max-h-[90vh]")
    expect(overlaySource).toContain("sm:max-w-[650px]")
    expect(overlaySource).toContain("radius-overlay")
  })

  it("uses the shared panel title role for dialog titles", () => {
    expect(source).toContain("textRole.panelTitle")
    expect(source).not.toContain("textRole.sectionTitle, className")
  })

  it("renders the default close control as the shared quiet icon button with focus-visible state", () => {
    expect(source).toContain('import { Button } from "@/components/ui/button"')
    expect(source).toContain('<DialogPrimitive.Close')
    expect(source).toContain('render={<Button type="button" variant="ghost" size="icon-sm" aria-label={tActions("close")} className="absolute top-4 right-4 opacity-70 hover:opacity-100"/>}')
    expect(source).toContain('<semanticIcons.action.cancel />')
    expect(source).toContain('import { semanticIcons } from "@/components/icons"')
    expect(source).not.toContain("focus:ring-")
    expect(source).not.toContain("focus:outline-hidden")
    expect(source).not.toContain("ring-offset-background")
  })
})
