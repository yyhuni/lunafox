import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/overview/vuln-severity-chart.tsx"), "utf8")

describe("vuln-severity-chart contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function VulnSeverityChart")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("consumes ordered shared severity ownership helpers", () => {
    expect(source).toContain("SEVERITY_LEVELS")
    expect(source).toContain("getSeverityColor")
  })
})
