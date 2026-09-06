import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/target/add-target-dialog-sections.tsx"), "utf8")

describe("add-target-dialog-sections contract", () => {
  it("uses the shared bulk line validation input shell", () => {
    expect(source).toContain("BulkLineValidationInput")
    expect(source).toContain("lineNumberedTextareaResponsiveViewportClassName")
    expect(source).toContain("viewportClassName={lineNumberedTextareaResponsiveViewportClassName}")
    expect(source).toContain("blockingIssueCount")
  })

  it("adapts target creation to the shared multi-select organization workspace", () => {
    expect(source).toContain("OrganizationSelectionWorkspace")
    expect(source).toContain('import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"')
    expect(source).toContain('defaultOpen={false}')
    expect(source).toContain('t("linkOrganization")')
    expect(source).toContain("ChevronRight")
    expect(source).toContain("group-data-[panel-open]:rotate-90")
    expect(source).toContain('CollapsibleContent className="pt-4"')
    expect(source).toContain('selectionMode="multiple"')
    expect(source).toContain("selectedOrganizationIds={formOrganizationIds}")
    expect(source).toContain('t("organizationWorkspaceTitle")')
    expect(source).toContain('t("organizationWorkspaceHint")')
    expect(source).toContain("onToggleOrganization={(organization) => onToggleOrganization(String(organization.id))}")
    expect(source).toContain("onClearOrganizations={onClearOrganizations}")
    expect(source).not.toContain("<table")
    expect(source).not.toContain("Popover")
    expect(source).not.toContain("Command")
  })

  it("does not retain a caller-local organization renderer", () => {
    expect(source).not.toContain("OrganizationScope")
    expect(source).not.toContain("OrganizationAvatar")
    expect(source).not.toContain("OrganizationWorkspacePagination")
    expect(source).not.toContain("SearchInput")
    expect(source).not.toContain("Checkbox")
    expect(source).not.toContain("OrganizationView")
    expect(source).not.toContain("setView")
  })

  it("keeps target-owned query controls and empty-state recovery", () => {
    expect(source).toContain("ORGANIZATION_WORKSPACE_PAGE_SIZE_OPTIONS = [10, 20, 50]")
    expect(source).toContain("onSearchQueryChange={setOrgSearchQuery}")
    expect(source).toContain("canPreviousPage={organizationPaginationNavigation.canPreviousPage}")
    expect(source).toContain("canNextPage={organizationPaginationNavigation.canNextPage}")
    expect(source).toContain("onPreviousPage={onPreviousOrgPage}")
    expect(source).toContain("onNextPage={onNextOrgPage}")
    expect(source).toContain("onPageSizeChange={setOrgPageSize}")
    expect(source).toContain("disabled={isPending}")
    expect(source).toContain('href="/organizations/"')
  })
})
