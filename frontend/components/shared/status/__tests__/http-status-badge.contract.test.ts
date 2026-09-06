import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "components/shared/status/http-status-badge.tsx"),
  "utf8"
)

describe("http-status-badge contract", () => {
  it("owns HTTP status badge rendering through shared status helpers", () => {
    expect(source).toContain("export type HttpStatusBadgeSize")
    expect(source).toContain("export function HttpStatusBadge")
    expect(source).toContain("getHttpStatusBadgeVariant")
    expect(source).toContain("getHttpStatusBadgeClassName")
    expect(source).toContain("textRole.tableCellSecondary")
    expect(source).toContain("font-mono")
    expect(source).toContain("tabular-nums")
  })

  it("keeps status badge density changes behind named size variants", () => {
    expect(source).toContain("sizeClassNames")
    expect(source).toContain('table: "h-5 px-2 py-0"')
    expect(source).toContain('default: "px-2 py-1"')
    expect(source).toContain('overlay: "px-1.5 py-0.5 text-xs backdrop-blur-sm"')
    expect(source).not.toContain("bg-green-500")
    expect(source).not.toContain("text-emerald")
  })
})
