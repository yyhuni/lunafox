import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/vulnerabilities/vulnerabilities-vertical-table.tsx"), "utf8")

describe("vulnerabilities-vertical-table contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function VulnerabilitiesVerticalTable")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("does not force severity colors with important utility overrides", () => {
    expect(source).not.toContain("!bg-[var(--sev-bg)]")
    expect(source).not.toContain("!text-[var(--sev-fg)]")
    expect(source).not.toContain("!border-[var(--sev-border)]")
  })

  it("uses shared severity badge variants instead of local css vars", () => {
    expect(source).toContain("variant={SEVERITY_VARIANTS[item.severity]}")
    expect(source).toContain("VULNERABILITY_SEVERITY_BADGE_CLASS")
    expect(source).toContain('useTranslations("vulnerabilities.severity")')
    expect(source).toContain("const severityLabel = tSeverity(item.severity")
    expect(source).not.toContain("item.severity.slice(0, 3)")
    expect(source).not.toContain('className="sr-only"')
    expect(source).not.toContain("w-[4ch]")
    expect(source).not.toContain("md:w-[5ch]")
    expect(source).not.toContain("hexToRgba")
    expect(source).not.toContain("--sev-fg")
    expect(source).not.toContain("bg-[var(--sev-bg)]")
  })

  it("uses shared status text helpers for review icons", () => {
    expect(source).toContain("getStatusToneTextClass")
    expect(source).not.toContain("text-green-600")
    expect(source).not.toContain("text-blue-500")
  })
})
