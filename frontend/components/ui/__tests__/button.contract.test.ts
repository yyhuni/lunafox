import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/button.tsx"), "utf8")
const readme = readFileSync(path.resolve(process.cwd(), "components/ui/README.md"), "utf8")
const outlineVariant = source.slice(source.indexOf("outline:"), source.indexOf("surface:"))
const surfaceVariant = source.slice(source.indexOf("surface:"), source.indexOf("secondary:"))

describe("button contract", () => {
  it("uses theme-driven radius without hover motion for primary actions", () => {
    expect(source).toContain("radius-control")
    expect(source).not.toContain("hover:border-b-2")
    expect(source).not.toContain("hover:-translate-y-px")
  })

  it("keeps generic dropdown triggers on their semantic button variant", () => {
    expect(source).not.toContain("data-[popup-open]:bg-muted")
    expect(readme).toContain("Opening a generic `DropdownMenu` must not replace a `Button`'s semantic variant colors")
  })

  it("adds semantic button aliases without raw var highlight utilities", () => {
    expect(source).toContain("primary:")
    expect(source).toContain("bg-interaction-accent text-interaction-accent-foreground")
    expect(source).toContain("hover:bg-interaction-accent/90")
    expect(source).toContain('link: "text-primary underline-offset-4 hover:text-interaction-accent hover:underline"')
    expect(source).toContain("surface:")
    expect(source).toContain("quiet:")
    expect(source).not.toContain("hover:border-b-[var(--highlight)]")
    expect(source).not.toContain("hover:border-b-highlight")
  })

  it("keeps surface buttons opaque while outline buttons retain their dark translucent treatment", () => {
    expect(surfaceVariant).toContain("bg-background")
    expect(surfaceVariant).not.toContain("dark:bg-input/30")
    expect(surfaceVariant).not.toContain("dark:hover:bg-input/50")
    expect(outlineVariant).toContain("dark:bg-input/30")
    expect(readme).toContain("`surface` remains opaque in both light and dark themes")
  })

  it("keeps the near-code button system documentation in sync", () => {
    expect(readme).toContain("Button System")
    expect(readme).toContain("default")
    expect(readme).toContain("sm")
    expect(readme).toContain("action-card")
    expect(readme).toContain("icon-sm")
    expect(readme).toContain("Primary/default buttons must keep stable geometry on hover")
    expect(readme).toContain("Do not hand-code primary production buttons")
    expect(readme).toContain("button-standardization.contract.test.ts")
  })

  it("owns the shared button loading state contract", () => {
    expect(source).toContain("\"action-card\":")
    expect(source).toContain("loading?: boolean")
    expect(source).toContain("loadingLabel?: React.ReactNode")
    expect(source).toContain("data-loading")
    expect(source).toContain("aria-busy")
    expect(source).toContain("button-loading-indicator")
    expect(source).toContain("disabled || loading")
  })

  it("exposes selected density for compact control-bar guardrails", () => {
    expect(source).toContain('"data-size": size ?? "default"')
  })

  it("owns common layout affordances that callers used to patch with local sizing classes", () => {
    expect(source).toContain("layout?:")
    expect(source).toContain("data-layout")
    expect(source).toContain("fullWidth")
    expect(source).toContain("between")
    expect(source).toContain("tableHeader")
    expect(source).toContain("tableHeaderInline")
    expect(source).toContain("data-[popup-open]:bg-primary/10")
    expect(source).toContain("data-[popup-open]:text-primary")
    expect(source).toContain("textCellLink")
    expect(source).toContain("actionTile")
    expect(source).toContain("data-selected")
    expect(source).toContain("selected?: boolean")
  })

  it("uses Base UI button and project polymorphic composition instead of Radix Slot", () => {
    expect(source).toContain('from "@base-ui/react/button"')
    expect(source).toContain('from "@/components/ui/polymorphic"')
    expect(source).not.toContain("@radix-ui/react-slot")
  })
})
