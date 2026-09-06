import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/vulnerabilities/vulnerabilities-vertical-resize-handle.tsx"), "utf8")

describe("vulnerabilities-vertical-resize-handle contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function VerticalResizeHandle")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("renders as an in-flow gap handle instead of an absolute overlay", () => {
    expect(source).toContain("h-2 shrink-0 cursor-row-resize")
    expect(source).not.toContain("absolute")
    expect(source).not.toContain("top-0")
    expect(source).not.toContain("bottom-0")
    expect(source).not.toContain('position?: "top" | "bottom"')
  })
})
