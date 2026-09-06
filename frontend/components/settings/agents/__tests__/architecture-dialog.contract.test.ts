import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/architecture-dialog.tsx"), "utf8")

describe("architecture-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ArchitectureDialog")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/ui/dialog\"")
  })

  it("renders the architecture command center without tab navigation", () => {
    expect(source).toContain("ArchitectureCommandCenter")
    expect(source).not.toContain("ArchitectureDialogStats")
    expect(source).toContain("max-w-screen-2xl")
    expect(source).toContain('width: "min(1184px, calc(100vw - 48px))"')
    expect(source).toContain('height: "min(720px, calc(100vh - 48px))"')
    expect(source).not.toContain("@/components/ui/tabs")
    expect(source).not.toContain("<Tabs")
  })

  it("reserves a right-side gutter for the dialog close button", () => {
    expect(source).toContain("pl-6 pr-14 py-5")
    expect(source).not.toContain("border-b px-6 py-5")
  })
})
