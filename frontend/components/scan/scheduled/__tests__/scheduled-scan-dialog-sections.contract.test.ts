import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/scheduled-scan-dialog-sections.tsx"), "utf8")
const createDialogSource = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/create-scheduled-scan-dialog.tsx"), "utf8")

describe("scheduled-scan-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ScheduledScanScopeStep")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("keeps scan scope as the first drawer step with organization and target choices", () => {
    expect(source).toContain("interface ScheduledScanScopeStepProps")
    expect(source).toContain('t("form.scanScope")')
    expect(source).toContain('t("form.organizationScan")')
    expect(source).toContain('t("form.targetScan")')
    expect(source).toContain('t("form.selectOrganization")')
    expect(source).toContain('t("form.selectTarget")')
  })

  it("adapts organization scope to the shared single-select workspace", () => {
    expect(source).toContain("OrganizationSelectionWorkspace")
    expect(source).toContain('selectionMode="single"')
    expect(source).toContain('showSelectionCount={false}')
    expect(source).toContain('id="scheduled-scan-organization-workspace"')
    expect(source).toContain("selectedOrganizationIds={selectedOrgId === null ? [] : [String(selectedOrgId)]}")
    expect(source).toContain("onToggleOrganization={(organization) => onSelectOrg(organization.id)}")
    expect(source).toContain("onClearOrganizations={() => setSelectedOrgId(null)}")
    expect(source).not.toContain("Popover")
    expect(source).not.toContain("ChevronsUpDown")
  })

  it("keeps organization query state in the scheduled-scan caller", () => {
    expect(source).toContain("SCHEDULED_SCAN_ORGANIZATION_PICKER_PAGE_SIZE_OPTIONS")
    expect(source).toContain("organizationPaginationNavigation")
    expect(source).toContain("onSearchQueryChange={setOrgSearchInput}")
    expect(source).toContain("onPreviousPage={onPreviousOrgPage}")
    expect(source).toContain("onNextPage={onNextOrgPage}")
    expect(source).toContain("onPageSizeChange={setOrgPageSize}")
    expect(createDialogSource).toContain("onPreviousOrgPage={onPreviousOrgPage}")
    expect(createDialogSource).toContain("onNextOrgPage={onNextOrgPage}")
    expect(createDialogSource).toContain("setOrgPageSize={setOrgPageSize}")
    expect(createDialogSource).not.toContain('useTranslations("common.pagination")')
  })

  it("adapts target scope to the target-domain single-select workspace", () => {
    expect(source).toContain("TargetSelectionWorkspace")
    expect(source).toContain('id="scheduled-scan-target-workspace"')
    expect(source).toContain("selectedTargetId={selectedTargetId}")
    expect(source).toContain("canPreviousPage={targetPaginationNavigation.canPreviousPage}")
    expect(source).toContain("canNextPage={targetPaginationNavigation.canNextPage}")
    expect(source).toContain("onPreviousPage={onPreviousTargetPage}")
    expect(source).toContain("onNextPage={onNextTargetPage}")
    expect(source).toContain("onToggleTarget={(target) => onSelectTarget(target.id)}")
    expect(source).toContain("onClearTarget={() => setSelectedTargetId(null)}")
    expect(source).not.toContain("CommandList")
    expect(source).not.toContain("CommandItem")
    expect(source).not.toContain('t("selectedCount"')
  })

  it("bounds long scope labels inside the drawer instead of widening it", () => {
    expect(source).toContain("min-w-0")
    expect(source).toContain("max-w-full")
    expect(source).toContain("truncate")
    expect(source).toContain("overflow-hidden")
    expect(source).toContain("flex-1")
  })

  it("keeps scheduled-only scope and schedule sections separate from reused scan controls", () => {
    expect(source).toContain("export function ScheduledScanScopeStep")
    expect(source).toContain("export function ScheduledScanScheduleStep")
    expect(source).toContain("className?: string")
    expect(source).not.toContain("export function ScheduledScanWorkflowStep")
    expect(source).not.toContain("export function ScheduledScanConfigStep")
    expect(createDialogSource).toContain("InitiateScanWorkflowSelection")
    expect(createDialogSource).toContain("InitiateScanConfigStep")
    expect(createDialogSource).toContain("ScanAgentSelector")
    expect(source).not.toContain("export function ScheduledScanTargetSelectionStep")
  })

  it("keeps the fourth drawer step UTC-only", () => {
    expect(source).not.toContain("ScheduledScanTimeZoneField")
    expect(source).not.toContain("time-zone")
    expect(source).toContain("getNextExecutions(cronExpression)")
    expect(createDialogSource).toContain("const totalSteps = 4")
    expect(createDialogSource).not.toContain("timeZone={timeZone}")
    expect(createDialogSource).toMatch(/currentStep === 4[\s\S]*ScheduledScanScheduleStep/)
  })
})
