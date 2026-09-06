import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/quick-scan-dialog.tsx"), "utf8")

describe("quick-scan-dialog contract", () => {
  it("exposes stable smoke selectors for trigger and drawer shell", () => {
    expect(source).toContain("export function QuickScanDialog")
    expect(source).toContain("data-slot=\"quick-scan-trigger\"")
    expect(source).toContain("data-slot=\"quick-scan-drawer\"")
    expect(source).not.toContain("data-slot=\"quick-scan-dialog\"")
  })

  it("reuses the scan initiation workbench drawer shell", () => {
    expect(source).toContain("DrawerContent")
    expect(source).toContain("swipeDirection=\"right\"")
    expect(source).toContain("scanWorkbenchDrawerContentClassName")
    expect(source).toContain("<form className=\"flex h-full min-h-0 w-full flex-col\">")
    expect(source).toContain("InitiateScanStepHeader")
    expect(source).toContain("shrink-0 border-t bg-card px-5 py-4 sm:px-6")
    expect(source).not.toContain("DialogContent")
    expect(source).not.toContain("max-w-[90vw]")
  })

  it("derives header progress from the current scan step only", () => {
    expect(source).toContain("getScanStepProgress")
    expect(source).toContain("const stepProgress = getScanStepProgress(step, totalSteps)")
  })

  it("reuses initiate-scan workflow and engine configuration steps", () => {
    expect(source).toContain("InitiateScanWorkflowSelection")
    expect(source).toContain("InitiateScanExecutionOptions")
    expect(source).toContain("InitiateScanConfigStep")
    expect(source).toContain("InitiateScanFooter")
    expect(source).toContain("InitiateScanOverwriteDialog")
    expect(source).toContain("QuickScanFooter")
    expect(source).not.toContain("ScanWorkflowProfileSelector")
    expect(source).not.toContain("ScanConfigEditor")
    expect(source).not.toContain("QuickScanOverwriteDialog")
  })

  it("keeps the quick-scan footer aware of its outer step sequence", () => {
    expect(source).toContain("currentStep={step - 1}")
    expect(source).toContain("showBackButton={step > 1}")
    expect(source).toContain("selectedWorkflowDisplayName={selectedWorkflowDisplayName}")
  })

  it("avoids layout-shifting trigger hover motion", () => {
    expect(source).not.toContain("group-hover:translate-x")
  })
})
