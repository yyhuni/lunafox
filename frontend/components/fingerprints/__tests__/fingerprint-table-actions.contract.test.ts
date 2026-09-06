import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/fingerprints/fingerprint-table-actions.tsx"), "utf8")

describe("fingerprint-table-actions contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useFingerprintTableActions")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("routes short fingerprint toolbar menus through the shared toolbar owner", () => {
    expect(source).toContain('from "@/components/shared/data-table"')
    expect(source).toContain("ToolbarActionMenu")
    expect(source).not.toContain('className="w-48"')
    expect(source).not.toContain('className="w-40"')
  })

  it("routes selected delete through selected-row actions instead of the toolbar menu", () => {
    expect(source).toContain("selectedRowActions")
    expect(source).toContain('label: tCommon("delete")')
    expect(source).toContain('tone: "destructive"')
    expect(source).toContain("setBulkDeleteDialogOpen(true)")
    expect(source).not.toContain('{tCommon("delete")} ({selectedCount})')
    expect(source).not.toContain('{t("actions.deleteSelected")} ({selectedCount})')
  })
})
