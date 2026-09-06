import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/severity-config.ts"), "utf8")

describe("severity-config contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function getSeverityStyle")
  })

  it("exports ordered severity ownership helpers for shared chart consumers", () => {
    expect(source).toContain("export const SEVERITY_LEVELS")
    expect(source).toContain("export function getSeverityColor")
    expect(source).toContain("SEVERITY_TEXT_CLASSNAMES")
    expect(source).toContain("SEVERITY_CARD_STYLES")
  })

  it("routes severity colors through theme tokens instead of fixed source colors", () => {
    for (const token of [
      "--severity-critical",
      "--severity-high",
      "--severity-medium",
      "--severity-low",
      "--severity-info",
      "--severity-critical-background",
    ]) {
      expect(source).toContain(token)
    }

    for (const className of [
      "text-severity-critical",
      "bg-severity-critical-bg",
      "border-severity-critical-border",
      "hover:bg-severity-critical-hover",
    ]) {
      expect(source).toContain(className)
    }

    expect(source).not.toMatch(/#[0-9a-fA-F]{3,8}/)
    expect(source).not.toMatch(/rgba?\(/)
    expect(source).not.toContain("SEVERITY_COLORS_DARK")
  })
})
