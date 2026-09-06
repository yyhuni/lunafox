import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/editors/codemirror-theme.ts"), "utf8")

describe("codemirror-theme contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"@codemirror/view\"")
  })

  it("owns editor colors through LunaFox theme tokens", () => {
    expect(source).toContain("var(--card)")
    expect(source).toContain("var(--foreground)")
    expect(source).toContain("var(--muted-foreground)")
    expect(source).toContain("var(--primary)")
    expect(source).toContain(".cm-selectionBackground")
  })
})
