import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/badge.tsx"), "utf8")

describe("badge contract", () => {
  it("keeps badge-type semantics in component source", () => {
    expect(source).toContain("data-[badge-type=subdomain]:text-muted-foreground")
    expect(source).toContain("data-[badge-type=workflow]:text-muted-foreground")
    expect(source).toContain("data-[badge-type=engine]:text-muted-foreground")
    expect(source).toContain("isSharedScanStatus")
    expect(source).toContain("getScanStatusBadgeVariant")
    expect(source).toContain("getScanStatusColorVar")
    expect(source).toContain('return "border-l-[3px] pl-2"')
  })

  it("consumes shared status and severity ownership helpers", () => {
    expect(source).toContain('from "@/lib/status-config"')
    expect(source).toContain('from "@/lib/severity-config"')
    expect(source).toContain("radius-badge")
    expect(source).toContain('count: "radius-pill')
    expect(source).toContain("count:")
    expect(source).toContain("filterCount:")
    expect(source).not.toContain("#9b1c31")
    expect(source).not.toContain("#dc2626")
    expect(source).not.toContain("rgba(")
    expect(source).not.toContain("text-[var(--success)]")
    expect(source).not.toContain("bg-[var(--warning)]")
  })

  it("keeps polymorphic composition project-owned instead of using Radix Slot", () => {
    expect(source).toContain('from "@/components/ui/polymorphic"')
    expect(source).not.toContain("@radix-ui/react-slot")
  })
})
