import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "components/shared/data-table/mono-value-cell.tsx"),
  "utf8"
)

describe("mono-value-cell contract", () => {
  it("owns dense business-list metric typography and box height", () => {
    expect(source).toContain("export function MonoValueCell")
    expect(source).toContain("textRole.tableCellSecondary")
    expect(source).toContain("min-h-5")
    expect(source).toContain("items-center")
    expect(source).toContain("whitespace-nowrap")
    expect(source).toContain("font-mono")
  })
})
