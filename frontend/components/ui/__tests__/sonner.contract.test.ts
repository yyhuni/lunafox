import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/sonner.tsx"), "utf8")

describe("sonner contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/icons\"")
    expect(source).toContain("useColorTheme")
  })

  it("maps toaster shell colors to semantic card tokens", () => {
    expect(source).toContain('"--normal-bg": "var(--card)"')
    expect(source).toContain('"--normal-text": "var(--card-foreground)"')
    expect(source).toContain('"--normal-border": "var(--border)"')
    expect(source).toContain('"--border-radius": "var(--radius-overlay)"')
  })

  it("keeps transient feedback on the shared compact surface tier", () => {
    expect(source).toContain('width: "fit-content"')
    expect(source).toContain('marginInline: "auto"')
    expect(source).toContain('padding: "8px 12px"')
    expect(source).toContain('gap: "8px"')
    expect(source).toContain('borderRadius: "var(--radius-control)"')
    expect(source).toContain('title: "!text-xs !leading-4"')
    expect(source).toContain('description: "!text-xs !leading-4"')
  })

  it("collapses stacked toasts until the user interacts with the stack", () => {
    expect(source).toContain("expand={false}")
  })

  it("preserves Sonner's per-toast stacking order", () => {
    expect(source).not.toContain("zIndex: 100")
  })

  it("keeps global toast stacking bounded and readable", () => {
    expect(source).toContain("visibleToasts={3}")
    expect(source).toContain("duration={4000}")
    expect(source).toContain("gap={12}")
  })

  it("follows the active LunaFox theme instead of a hard-coded toast theme", () => {
    expect(source).toContain('theme={currentTheme.isDark ? "dark" : "light"}')
    expect(source).not.toContain('theme="light"')
  })
})
