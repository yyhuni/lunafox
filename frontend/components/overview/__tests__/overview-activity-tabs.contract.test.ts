import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/overview/overview-activity-tabs.tsx"), "utf8")

describe("overview-activity-tabs contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function OverviewActivityTabs")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/ui/tabs\"")
  })
})
