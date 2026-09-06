import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/visualization/mermaid-diagram.tsx"), "utf8")

describe("mermaid-diagram contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function MermaidDiagram")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })
})
