import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/create-scheduled-scan-dialog.tsx"), "utf8")

describe("create-scheduled-scan-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function CreateScheduledScanSheet")
    expect(source).toContain("export const CreateScheduledScanDialog = CreateScheduledScanSheet")
    expect(source).toContain("SheetContent")
    expect(source).toContain("className")
    expect(source).toContain("scanWorkbenchDrawerContentClassName")
    expect(source).not.toContain("detailDrawerBackdropClassName")
    expect(source).not.toContain("overlayClassName")
    expect(source).not.toContain("sideMotion")
    expect(source).toContain("from \"react\"")
    expect(source).not.toContain("DialogContent")
  })

  it("keeps create scheduled scan sheets on the shared workbench drawer motion contract", () => {
    expect(source).toContain('from "@/lib/ui/overlay-styles"')
    expect(source).toMatch(/<SheetContent[\s\S]*side="right"[\s\S]*className=\{scanWorkbenchDrawerContentClassName\}/)
    expect(source).not.toContain("overlayBackdropClassName")
    expect(source).not.toContain('className="flex flex-col gap-0 overflow-hidden p-0 w-full sm:max-w-xl lg:max-w-3xl xl:max-w-4xl"')
  })

  it("uses the canonical local-dismiss icon for the sheet close control", () => {
    expect(source).toContain("semanticIcons.action.cancel")
    expect(source).not.toContain("XIcon")
  })

  it("keeps scheduled scan creation as a four-step workbench flow", () => {
    expect(source).toContain("const totalSteps = 4")
    expect(source).not.toContain("hasPreset ? 4 : 5")
    expect(source).toContain("ScheduledScanScopeStep")
    expect(source).not.toContain("ScheduledScanBasicInfoStep")
    expect(source).not.toContain("ScheduledScanTargetSelectionStep")
  })

  it("keeps scheduling separate from scan scope and reuses the quick-scan selection and options steps", () => {
    expect(source).toMatch(/currentStep === 1[\s\S]*ScheduledScanScopeStep/)
    expect(source).toMatch(/currentStep === 2[\s\S]*InitiateScanWorkflowSelection[\s\S]*ScanAgentSelector/)
    expect(source).toMatch(/currentStep === 3[\s\S]*InitiateScanConfigStep/)
    expect(source).toMatch(/currentStep === 4[\s\S]*ScheduledScanScheduleStep/)
    expect(source).toContain('from "@/components/scan/initiate-scan-dialog-sections"')
    expect(source).not.toContain("ScheduledScanWorkflowStep")
    expect(source).not.toContain("ScheduledScanConfigStep")
    expect(source).not.toContain("currentStep === 5")
  })
})
