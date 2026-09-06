import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/tools/commands/commands-data-table.tsx"), "utf8")

describe("commands-data-table contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function CommandsDataTable")
    expect(source).toContain("BusinessListDataTable")
  })

  it("uses the standard business-list route-facing wrapper", () => {
    expect(source).toContain('expandColumnIds: ["displayName"]')
    expect(source).toContain('toolbarDensity="compact"')
    expect(source).not.toContain("UnifiedDataTable")
  })
})
