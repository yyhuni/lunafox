import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/organization-columns.tsx"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "components/organization/organization-table-layout.ts"), "utf8")

describe("organization-columns contract", () => {
  it("only exposes backend-backed sorting for organization name and created time", () => {
    expect(source).toContain('orderBy: "displayName"')
    expect(source).toContain('orderBy: "createdAt"')
    expect(source).toContain("serverSortPerformance")
    expect(source).toContain("firstSortDirection")
  })

  it("keeps non-backed organization columns unsortable", () => {
    expect(source).toContain('accessorKey: "targetCount"')
    expect(source).toContain("enableSorting: false")
    expect(source).not.toContain('orderBy: "targetCount"')
  })

  it("keeps the stacked description preview within the comfortable row rhythm", () => {
    expect(source).toContain("row.original.description")
    expect(source).toContain("maxLines={1}")
    expect(source).toContain("textRoleName=\"tableCellSecondary\"")
    expect(source).not.toContain('accessorKey: "description"')
    expect(source).not.toContain("organizationTableColumnLayout.description")
  })

  it("renders a language-neutral initial marker without a filled background", () => {
    expect(source).toContain("getOrganizationInitial")
    expect(source).toContain('aria-hidden="true"')
    expect(source).toContain("border-dashed")
    expect(source).toContain("border-solid")
    expect(source).toContain("bg-transparent")
    expect(source).toContain("group-hover:border-solid")
    expect(source).toContain("group-data-[state=selected]:border-solid")
    expect(source).not.toContain("bg-muted")
    expect(source).not.toContain("OrganizationIcon")
  })

  it("keeps the organization scan action visible beside the overflow menu", () => {
    expect(source).toContain('id: "actions"')
    expect(source).toContain("OrganizationRowActions")
    expect(source).toContain("DenseRowActionButton")
    expect(source).toContain("leadingActions")
    expect(source).toContain("label={t.tooltips.initiateScan}")
    expect(source).not.toContain("<DropdownMenuItem onClick={onInitiateScan}>")
    expect(source).not.toContain("stickyRight")
    expect(source).toContain('className="flex justify-end"')
    expect(layoutSource).toContain('actions: { key: "actions", size: 88, minSize: 88, maxSize: 88 }')
  })

  it("keeps the organization identity marker legible while its row is hovered or selected", () => {
    expect(source).toContain("group-data-[state=selected]:text-foreground")
    expect(source).toContain("group-hover:text-foreground")
    expect(source).not.toContain("group-data-[state=selected]:bg-primary")
    expect(source).not.toContain("group-data-[state=selected]:text-primary")
  })
})
