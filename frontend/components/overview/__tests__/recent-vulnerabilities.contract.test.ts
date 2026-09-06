import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/overview/recent-vulnerabilities.tsx"), "utf8")

describe("recent-vulnerabilities contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function RecentVulnerabilities")
    expect(source).toContain("className")
    expect(source).toContain("import { useRecentVulnerabilities } from \"@/hooks/use-vulnerabilities\"")
    expect(source).toContain("const { data, isLoading } = useRecentVulnerabilities(5)")
    expect(source).not.toContain("from \"@tanstack/react-query\"")
  })

  it("uses shared severity and review status helpers", () => {
    expect(source).toContain("getSeverityVariant")
    expect(source).toContain("getStatusToneBadgeClass")
    expect(source).not.toContain("const severityConfig")
    expect(source).not.toContain("SEVERITY_STYLES")
    expect(source).not.toContain("bg-blue-500/10")
    expect(source).not.toContain("text-blue-600")
  })

  it("uses the shared dense table rhythm for overview vulnerability rows", () => {
    expect(source).toContain("TABLE_DENSE_ROW_CLASS")
    expect(source).toContain("TABLE_DENSE_CELL_RHYTHM_CLASS")
    expect(source).toContain('className="h-12 w-full"')
    expect(source).not.toContain('className="h-10 w-full"')
  })
})
