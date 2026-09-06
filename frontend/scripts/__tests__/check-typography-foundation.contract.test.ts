import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const scriptPath = path.resolve(process.cwd(), "scripts/check-typography-foundation.mjs")

describe("check-typography-foundation contract", () => {
  it("provides a dedicated typography guardrail script", () => {
    expect(existsSync(scriptPath)).toBe(true)
  })

  it("locks full production strong-signal patterns and targeted strict typography patterns", () => {
    if (!existsSync(scriptPath)) {
      expect.fail("scripts/check-typography-foundation.mjs is missing")
    }

    const source = readFileSync(scriptPath, "utf8")

    expect(source).toContain("const fullProductionRoots")
    expect(source).toContain("components")
    expect(source).toContain("app")
    expect(source).toContain("const targetedScanRoots")
    expect(source).toContain("components/ui/table.tsx")
    expect(source).toContain("components/ui/badge.tsx")
    expect(source).toContain("components/ui/sidebar.tsx")
    expect(source).toContain("components/vulnerabilities/vulnerabilities-vertical-table.tsx")
    expect(source).toContain("components/search/search-page-sections.tsx")
    expect(source).toContain("components/shared/data-table/pagination.tsx")
    expect(source).toContain("components/scan/workflow-profile-selector.tsx")
    expect(source).toContain("components/settings/agents/agent-list.tsx")
    expect(source).toContain("components/endpoints/endpoints-columns.tsx")
    expect(source).toContain("components/target/target-overview-sections.tsx")

    expect(source).toContain("text-foreground/80")
    expect(source).toContain("tracking-[0.14em]")
    expect(source).toContain("tracking-[0.12em]")
    expect(source).toContain("tracking-[0.2em] uppercase")
    expect(source).toContain("text-[9px]")
    expect(source).toContain("text-[10px]")
    expect(source).toContain("decorative-uppercase")
    expect(source).toContain("text-muted-foreground text-sm")
    expect(source).toContain("text-muted-foreground text-xs")
    expect(source).toContain("font-medium text-sm")
    expect(source).toContain("font-semibold text-sm")
    expect(source).toContain("font-medium text-lg")
    expect(source).toContain('scope: "full"')
    expect(source).toContain('scope: "targeted"')
  })

  it("supports inventory and verify modes and documents allowed exceptions", () => {
    if (!existsSync(scriptPath)) {
      expect.fail("scripts/check-typography-foundation.mjs is missing")
    }

    const source = readFileSync(scriptPath, "utf8")

    expect(source).toContain("\"inventory\"")
    expect(source).toContain("\"verify\"")
    expect(source).toContain("terminal")
    expect(source).toContain("chart")
    expect(source).toContain("github-star-button.tsx")
    expect(source).not.toContain("bauhaus-page-header.tsx")
    expect(source).not.toContain("bauhaus-overview-header.tsx")
    expect(source).toContain("prototype")
    expect(source).toContain("demo")
    expect(source).not.toContain('  "app-sidebar.tsx",')
  })
})
