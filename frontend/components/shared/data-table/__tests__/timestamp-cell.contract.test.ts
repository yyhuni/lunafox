import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "components/shared/data-table/timestamp-cell.tsx"),
  "utf8"
)

describe("timestamp-cell contract", () => {
  it("owns dense business-list timestamp typography and box height", () => {
    expect(source).toContain("export function TimestampCell")
    expect(source).toContain("textRole.tableCellSecondary")
    expect(source).toContain("min-h-5")
    expect(source).toContain("items-center")
    expect(source).toContain("whitespace-nowrap")
  })
})
