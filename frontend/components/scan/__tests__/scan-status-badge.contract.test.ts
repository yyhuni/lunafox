import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scan-status-badge.tsx"), "utf8")

describe("scan-status-badge contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared scan status color ownership instead of a local color switch", () => {
    expect(source).toContain("from \"@/lib/status-config\"")
    expect(source).not.toContain("const getStatusColor")
  })

  it("keeps all badge variants under theme radius control", () => {
    expect(source).toContain('const roundedClass = "rounded-lg"')
    expect(source).not.toContain('variant === "sharp" ? "rounded-none"')
  })

  it("keeps compact progress bars on transform-based motion", () => {
    expect(source).toContain("origin-left")
    expect(source).toContain("scaleX")
    expect(source).not.toContain("transition-[width")
    expect(source).not.toContain("width: `${displayProgress}%`")
  })

  it("keeps compact status icons on the canonical status and running-motion path", () => {
    expect(source).toContain('variant === "icon-only"')
    expect(source).toContain('role="img"')
    expect(source).toContain('aria-label={label}')
    expect(source).toContain('status === "running" && "animate-spin"')
    expect(source).toContain("semanticIcons.status.failed")
    expect(source).toContain("semanticIcons.status.cancelled")
    expect(source).not.toContain("IconCircleX")
  })
})
