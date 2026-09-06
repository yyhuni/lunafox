import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/quick-scan-dialog-state.ts"), "utf8")

describe("quick-scan-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useQuickScanDialogState")
    expect(source).toContain("from \"react\"")
  })

  it("loads the selected workflow's singleton Profile", () => {
    expect(source).toContain("selectedWorkflowNames")
    expect(source).toContain("handleScanWorkflowNameChange")
    expect(source).toContain("useLoadScanWorkflowProfile")
    expect(source).toContain("loadScanWorkflowProfile(workflowName)")
    expect(source).toContain("const draft = adaptWorkflowProfile(profile, workflow)")
    expect(source).toContain("serializeWorkflowProfileDraft(draft)")
    expect(source).not.toContain("buildConfigFromWorkflowDetails")
    expect(source).not.toContain("selectedWorkflowIds")
    expect(source).not.toContain("selectedPresetId")
    expect(source).not.toContain("handleWorkflowIdsChange")
  })
})
