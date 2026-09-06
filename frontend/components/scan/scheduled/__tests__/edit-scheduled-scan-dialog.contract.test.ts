import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/edit-scheduled-scan-dialog.tsx"), "utf8")

describe("edit-scheduled-scan-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function EditScheduledScanDialog")
    expect(source).toContain("useEditScheduledScanDialogState")
    expect(source).toContain("from \"react\"")
  })

  it("uses the shared right-side form drawer instead of a centered dialog", () => {
    expect(source).toContain('from "@/components/shared/form-drawer"')
    expect(source).toContain("<FormDrawer")
    expect(source).not.toContain("<Dialog")
    expect(source).not.toContain("DialogContent")
  })

  it("reuses the scan-launch workflow selector instead of a local radio list", () => {
    expect(source).toContain("InitiateScanWorkflowSelection")
    expect(source).toContain('useTranslations("scan.initiate")')
    expect(source).toContain("handleWorkflowNamesChange")
    expect(source).not.toContain("EditScheduledScanWorkflowSection")
  })

  it("groups edit fields and configuration in two internal drawer tabs", () => {
    expect(source).toContain('from "@/components/ui/tabs"')
    expect(source).toContain("InitiateScanConfigStep")
    expect(source).toContain('value="basic"')
    expect(source).toContain('value="configuration"')
    expect(source).toContain('bodyClassName="flex min-h-0 flex-col overflow-hidden"')
    expect(source).not.toContain("Sidebar")
  })

  it("keeps optional execution controls in the shared collapsed disclosure", () => {
    const basicTabSource = source.slice(
      source.indexOf('<TabsContent value="basic"'),
      source.indexOf('<TabsContent value="configuration"'),
    )

    expect(source).toContain("InitiateScanExecutionOptions")
    expect(basicTabSource).toContain("<InitiateScanExecutionOptions")
    expect(basicTabSource).toContain("inputSource={inputSource}")
    expect(basicTabSource).toContain("selectedAgentID={selectedAgentID}")
    expect(basicTabSource).toContain("<InitiateScanWorkflowSelection")
    expect(basicTabSource).toContain("<EditScheduledScanTargetSection")
    expect(basicTabSource).toContain("<EditScheduledScanCronSection")
    expect(basicTabSource).not.toContain("<ScanInputSourceSelector")
    expect(basicTabSource).not.toContain("<ScanAgentSelector")
  })
})
