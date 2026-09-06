import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/scheduled-scan-data-table-state.ts"), "utf8")

describe("scheduled-scan-data-table-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useScheduledScanDataTableState")
    expect(source).toContain("from \"next-intl\"")
  })
})
