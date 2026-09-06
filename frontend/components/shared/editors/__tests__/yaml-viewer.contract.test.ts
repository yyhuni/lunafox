import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/editors/yaml-viewer.tsx"), "utf8")

describe("yaml-viewer contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function YamlViewer")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses the shared codemirror theme adapter", () => {
    expect(source).toContain("from \"@/components/shared/editors/codemirror-theme\"")
    expect(source).toContain("codeMirrorLightTheme")
    expect(source).toContain("codeMirrorDarkTheme")
    expect(source).not.toContain("@codemirror/theme-one-dark")
    expect(source).not.toContain("oneDark")
  })
})
