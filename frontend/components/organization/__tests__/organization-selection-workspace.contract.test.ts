import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "components/organization/organization-selection-workspace.tsx"),
  "utf8"
)

describe("organization-selection-workspace contract", () => {
  it("owns one responsive organization card workspace", () => {
    expect(source).toContain("export function OrganizationSelectionWorkspace")
    expect(source).toContain('export type OrganizationSelectionMode = "single" | "multiple"')
    expect(source).toContain("selectionMode: OrganizationSelectionMode")
    expect(source).toContain("OrganizationWorkspacePagination")
    expect(source).toContain("canPreviousPage: boolean")
    expect(source).toContain("canNextPage: boolean")
    expect(source).toContain("onPreviousPage: () => void")
    expect(source).toContain("onNextPage: () => void")
    expect(source).not.toContain("getOrganizationPaginationItems")
    expect(source).not.toContain("ORGANIZATION_PAGE_WINDOW_SIZE")
    expect(source).not.toContain('item === "ellipsis"')
    expect(source).toContain('toolbarDensity="compact"')
    expect(source).toContain("showSelectionCount = true")
    expect(source).toContain("{showSelectionCount ? (")
    expect(source).toContain("sm:grid-cols-2")
    expect(source).toContain('className="border-t bg-muted/30 sm:h-18"')
    expect(source).toContain('<SelectTrigger size="sm" className="w-28">')
    expect(source).toContain('<Checkbox className="mt-2"')
    expect(source).toContain("isSelected && \"border-border\"")
    expect(source).toContain("isSelected && \"text-foreground\"")
    expect(source).not.toContain("<table")
  })

  it("stays controlled and leaves query and payload ownership to callers", () => {
    expect(source).toContain("selectedOrganizationIds: readonly string[]")
    expect(source).toContain("onSearchQueryChange")
    expect(source).toContain("onPreviousPage")
    expect(source).toContain("onNextPage")
    expect(source).toContain("onPageSizeChange")
    expect(source).toContain("onToggleOrganization")
    expect(source).toContain("onClearOrganizations")
    expect(source).not.toContain("useOrganizations")
    expect(source).not.toContain("useQuery")
    expect(source).not.toContain("organizationService")
    expect(source).not.toContain("organizationIds:")
    expect(source).not.toContain("organizationId:")
  })

  it("fast-fails invalid cardinality and pagination options", () => {
    expect(source).toContain('selectionMode === "single" && selectedOrganizationIds.length > 1')
    expect(source).toContain("pageSizeOptions.length === 0")
    expect(source).toContain("!Number.isFinite(size) || size <= 0")
  })
})
