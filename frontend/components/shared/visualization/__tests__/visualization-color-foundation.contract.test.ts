import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const waveGridSource = readFileSync(
  path.resolve(process.cwd(), "components/shared/visualization/wave-grid.tsx"),
  "utf8"
)
const mermaidSource = readFileSync(
  path.resolve(process.cwd(), "components/shared/visualization/mermaid-diagram.tsx"),
  "utf8"
)

describe("visualization color foundation", () => {
  it("renders wave grid opacity from resolved theme color without source-level rgba literals", () => {
    expect(waveGridSource).toContain("globalAlpha")
    expect(waveGridSource).not.toContain("`rgba(")
  })

  it("resolves Mermaid colors from CSS variables without legacy hsl token wrapping", () => {
    expect(mermaidSource).toContain("var(")
    expect(mermaidSource).not.toContain("`hsl(")
  })
})
