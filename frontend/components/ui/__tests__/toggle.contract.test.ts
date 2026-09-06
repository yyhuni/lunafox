import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/toggle.tsx"), "utf8")

describe("toggle contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
    expect(source).toContain("radius-control")
  })

  it("uses the Base UI toggle backend and Base UI pressed state markers", () => {
    expect(source).toContain('from "@base-ui/react/toggle"')
    expect(source).not.toContain("@radix-ui/react-toggle")
    expect(source).toContain("data-pressed:bg-accent")
    expect(source).not.toContain("data-[state=on]")
  })
})
