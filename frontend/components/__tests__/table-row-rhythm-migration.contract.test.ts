import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const directTableCallers = [
  "components/overview/recent-vulnerabilities.tsx",
  "components/vulnerabilities/vulnerabilities-vertical-table.tsx",
  "components/shared/data-table/unified-data-table.tsx",
]

const sources = directTableCallers.map((filePath) => ({
  filePath,
  source: readFileSync(path.resolve(process.cwd(), filePath), "utf8"),
}))

describe("table row rhythm migration contract", () => {
  it("keeps direct table callers on shared row rhythms", () => {
    for (const { filePath, source } of sources.filter(({ filePath }) => filePath !== "components/shared/data-table/unified-data-table.tsx")) {
      expect(source, filePath).toContain("TABLE_DENSE_ROW")
      expect(source, filePath).toContain("TABLE_DENSE_CELL")
      expect(source, filePath).toContain('data-row-rhythm="dense"')
      expect(source, filePath).not.toContain("--vuln-row-h")
      expect(source, filePath).not.toContain("--vuln-table-head-h")
    }

    const unifiedDataTable = sources.find(
      ({ filePath }) => filePath === "components/shared/data-table/unified-data-table.tsx"
    )
    expect(unifiedDataTable).toBeDefined()
    expect(unifiedDataTable?.source).toContain("TABLE_DENSE_ROW")
    expect(unifiedDataTable?.source).toContain("TABLE_DENSE_CELL")
    expect(unifiedDataTable?.source).toContain("TABLE_COMFORTABLE_ROW")
    expect(unifiedDataTable?.source).toContain("TABLE_COMFORTABLE_CELL")
    expect(unifiedDataTable?.source).toContain('const rowDensity = ui?.rowDensity ?? "dense"')
    expect(unifiedDataTable?.source).toContain("data-row-rhythm={rowDensity}")
    expect(unifiedDataTable?.source).not.toContain("--vuln-row-h")
    expect(unifiedDataTable?.source).not.toContain("--vuln-table-head-h")
  })

  it("keeps table row action buttons on shared icon-sm sizing", () => {
    const rowActionFiles = [
      "components/target/all-targets-columns.tsx",
      "components/organization/targets/targets-columns.tsx",
      "components/organization/organization-columns.tsx",
      "components/tools/commands/commands-columns.tsx",
    ]

    for (const filePath of rowActionFiles) {
      const source = readFileSync(path.resolve(process.cwd(), filePath), "utf8")

      expect(source, filePath).toMatch(/size="icon-sm"|DenseRowActionMenu/)
      expect(source, filePath).not.toContain('size="icon"')
      expect(source, filePath).not.toContain("h-8 p-0 w-8")
      expect(source, filePath).not.toContain("h-8 w-8")
    }

    const denseRowActionMenuSource = readFileSync(
      path.resolve(process.cwd(), "components/shared/data-table/menu-owners.tsx"),
      "utf8"
    )
    expect(denseRowActionMenuSource).toContain('triggerSize = "icon-sm"')
  })
})
