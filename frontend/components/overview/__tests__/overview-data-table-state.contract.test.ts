import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/overview/overview-data-table-state.ts"), "utf8")

describe("overview-data-table-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useOverviewDataTableState")
    expect(source).toContain("from \"react\"")
  })

  it("keeps overview status clicks wired to the existing progress dialog", () => {
    expect(source).toContain("handleStatusClick: handleViewProgress")
    expect(source).toContain('statusActionLabel: t("tooltips.viewProgress")')
  })
})
