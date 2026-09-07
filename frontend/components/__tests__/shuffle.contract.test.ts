import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shuffle.tsx"), "utf8")

describe("shuffle contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from 'react'")
  })

  it("keeps the static and animated line boxes aligned", () => {
    const css = readFileSync(path.resolve(process.cwd(), "components/shuffle.css"), "utf8")

    expect(css).toContain(".shuffle-parent")
    expect(css).toContain("line-height: 1.2;")
    expect(css).toContain(".shuffle-char")
    expect(css).toContain("line-height: inherit;")
    expect(css).not.toContain(".shuffle-char {\n  line-height: 1;")
  })
})
