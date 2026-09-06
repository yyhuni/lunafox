import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/pagination.tsx"), "utf8")

describe("pagination contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function DataTablePagination")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared compact control sizing for data-table pagination", () => {
    expect(source).toContain("<SelectTrigger\n              size=\"sm\"")
    expect(source).toContain('className="w-24"')
    expect(source).toContain('<SelectContent position="popper" width="content-fit">')
    expect(source).not.toContain("<SelectContent side=\"top\"")
    expect(source).toContain("size=\"icon-sm\"")
    expect(source).not.toContain("DATA_TABLE_PAGE_SIZE_TRIGGER_CLASSNAME")
    expect(source).not.toContain("className=\"h-8")
  })

  it("keeps pagination readable when controls wrap on mobile", () => {
    expect(source).toContain("flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between")
    expect(source).toContain("w-full sm:flex-1")
    expect(source).toContain("flex w-full flex-col gap-3 sm:w-auto sm:flex-row sm:items-center sm:gap-4")
    expect(source).toContain("textRole.metadataLabel")
    expect(source).toContain('"whitespace-nowrap", textRole.metadataLabel')
    expect(source).toContain("textRole.metadataValue")
    expect(source).not.toContain("textRole.helperText")
    expect(source).not.toContain("space-x-")
  })
})
