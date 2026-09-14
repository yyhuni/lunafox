import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/disk/disk-stat-cards.tsx"), "utf8")

describe("disk-stat-cards contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function DiskStatCards")
    expect(source).toContain("className")
    expect(source).toContain("from \"next-intl\"")
  })

  it("uses compact card chrome for the small resource summaries", () => {
    expect(source).toContain('<Card variant="compact" className="@container/card">')
  })
})
