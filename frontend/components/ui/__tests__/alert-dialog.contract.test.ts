import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/alert-dialog.tsx"), "utf8")

describe("alert-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared modal overlay style tokens", () => {
    expect(source).toContain("from \"@/lib/ui/overlay-styles\"")
    expect(source).toContain("overlayBackdropClassName")
    expect(source).toContain("centeredOverlayPanelClassName")
  })

  it("uses Base UI as the alert dialog primitive backend", () => {
    expect(source).toContain('from "@base-ui/react/alert-dialog"')
    expect(source).not.toContain("@radix-ui/react-alert-dialog")
    expect(source).toContain("AlertDialogPrimitive.Popup")
    expect(source).toContain("AlertDialogPrimitive.Backdrop")
  })

  it("hard-cuts action/cancel and asChild compatibility from the shared wrapper", () => {
    expect(source).not.toContain("asChild")
    expect(source).not.toContain("AlertDialogAction")
    expect(source).not.toContain("AlertDialogCancel")
    expect(source).toContain("AlertDialogClose")
    expect(source).toContain("AlertDialogPrimitive.Close")
  })

  it("uses the shared panel title role for alert dialog titles", () => {
    expect(source).toContain("textRole.panelTitle")
    expect(source).not.toContain("textRole.sectionTitle, className")
  })

  it("owns the destructive confirmation action variant", () => {
    expect(source).toContain('variant?: "default" | "destructive" | "outline"')
    expect(source).toContain("buttonVariants({ variant })")
    expect(source).toContain('variant === "destructive" && "dark:bg-destructive"')
  })
})
