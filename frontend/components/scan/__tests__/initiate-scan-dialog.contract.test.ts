import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/initiate-scan-dialog.tsx"), "utf8")
const legacySingleExport = ["Initiate", "Scan", "Sheet"].join("")
const legacyBulkExport = ["Bulk", "Initiate", "Scan", "Sheet"].join("")
const legacySingleAlias = ["Initiate", "Scan", "Dialog ="].join("")
const legacyBulkAlias = ["Bulk", "Initiate", "Scan", "Dialog ="].join("")

describe("initiate-scan-dialog contract", () => {
  it("exports only drawer component names for scan initiation", () => {
    expect(source).toContain("export function InitiateScanDrawer")
    expect(source).toContain("export function BulkInitiateScanDrawer")
    expect(source).not.toContain(legacySingleExport)
    expect(source).not.toContain(legacyBulkExport)
    expect(source).not.toContain(legacySingleAlias)
    expect(source).not.toContain(legacyBulkAlias)
  })

  it("uses the shared Base UI drawer contract", () => {
    expect(source).toContain("DrawerContent")
    expect(source).toContain("swipeDirection=\"right\"")
    expect(source).toContain("<DrawerClose")
    expect(source).toContain("render={")
    expect(source).toContain("scanWorkbenchDrawerContentClassName")
    expect(source).not.toContain("detailDrawerBackdropClassName")
    expect(source).not.toContain("overlayClassName")
    expect(source).not.toContain("SheetContent")
    expect(source).not.toContain("sideMotion=\"none\"")
    expect(source).not.toContain("createPortal")
  })

  it("uses a 640px right-side drawer width for the configuration workflow", () => {
    expect(source).toContain("data-[swipe-direction=right]:sm:max-w-[640px]")
    expect(source).not.toContain('"sm:max-w-xl lg:max-w-xl xl:max-w-xl"')
  })

  it("uses the shared scan icon container in the drawer header", () => {
    expect(source).toContain("EdgePanelHeader")
    expect(source).toContain('variant="workbench"')
    expect(source).toContain('<Play className="size-7" />')
    expect(source).toContain('<DrawerTitle className={textRole.sectionTitle}>')
    expect(source).toContain('cn("max-w-2xl", textRole.helperText)')
    expect(source).toContain("progress={<InitiateScanStepHeader")
  })

  it("uses the canonical local-dismiss icon for the drawer close control", () => {
    expect(source).toContain("semanticIcons.action.cancel")
    expect(source).not.toContain("XIcon")
  })

  it("derives header progress from the current scan step only", () => {
    expect(source).toContain("getScanStepProgress")
    expect(source).toContain("const stepProgress = getScanStepProgress(currentStep, steps.length)")
  })

  it("delegates accessibility and animation to Base UI Drawer", () => {
    expect(source).toContain("onOpenChange")
    expect(source).not.toContain('aria-modal="true"')
  })

  it("inherits the shared shadcn base-nova drawer backdrop", () => {
    expect(source).not.toContain("overlayBackdropBlur")
  })

  it("renders overwrite confirmation dialog independently", () => {
    expect(source).toContain("InitiateScanOverwriteDialog")
  })

  it("uses the shared optional execution-settings disclosure", () => {
    expect(source).toContain("InitiateScanExecutionOptions")
    expect(source).toContain("selectedAgentID")
    expect(source).toContain("inputSource")
  })

  it("shows the workflow display name in the first-step footer", () => {
    expect(source).toContain("const selectedWorkflowDisplayName = React.useMemo")
    expect(source).toContain("selectedWorkflow?.displayName || selectedWorkflow?.name")
    expect(source).toContain("selectedWorkflowDisplayName={selectedWorkflowDisplayName}")
  })
})
