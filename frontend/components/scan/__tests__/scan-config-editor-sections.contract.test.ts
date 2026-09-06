import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scan-config-editor-sections.tsx"), "utf8")

describe("scan-config-editor-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ScanConfigEditorLayout")
    expect(source).toContain("className")
    expect(source).toContain("from \"next-intl\"")
  })

  it("can visually hide the yaml field label under a shared section title", () => {
    expect(source).toContain("showLabel?: boolean")
    expect(source).toContain("showLabel = true")
    expect(source).toContain('labelClassName={showLabel ? undefined : "sr-only"}')
    expect(source).toContain('className="flex-1 overflow-hidden"')
    expect(source).toContain("fillHeight")
    expect(source).not.toContain("lineNumberedTextareaResponsiveViewportClassName")
    expect(source).not.toContain('className="flex-1 overflow-hidden px-4"')
    expect(source).not.toContain("console.debug")
  })
})
