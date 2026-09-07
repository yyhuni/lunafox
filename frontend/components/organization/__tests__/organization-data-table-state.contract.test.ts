import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/organization-data-table-state.ts"), "utf8")

describe("organization-data-table-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useOrganizationDataTableState")
    expect(source).toContain("from \"next-intl\"")
  })

  it("does not own table sorting defaults", () => {
    expect(source).not.toContain("defaultSorting")
    expect(source).toContain("useSimpleSearchState")
    expect(source).toContain("commitNow: commitSearch")
  })
})
