import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/initiate-scan-dialog-state.ts"), "utf8")

describe("initiate-scan-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useInitiateScanDialogState")
    expect(source).toContain("from \"react\"")
  })

  it("resets configuration from the selected workflow's current singleton Profile", () => {
    expect(source).toContain("const handleResetWorkflowConfig = useCallback")
    expect(source).toContain("const nextConfig = await loadProfileConfiguration(workflowName)")
    expect(source).toContain("const profile = await loadScanWorkflowProfile(workflowName)")
    expect(source).toContain("const draft = adaptWorkflowProfile(profile, workflow)")
    expect(source).toContain("serializeWorkflowProfileDraft(draft)")
    expect(source).toContain("setIsConfigEdited(false)")
    expect(source).toContain("handleResetWorkflowConfig")
  })

  it("exposes the explicit all-disabled state separately from YAML validity", () => {
    expect(source).toContain("const hasNoEnabledSteps = hasNoEnabledWorkflowSteps(configuration)")
    expect(source).toContain("hasNoEnabledSteps,")
    expect(source).toContain("!hasNoEnabledSteps")
  })

  it("keeps internal configuration sync separate from user edits", () => {
    expect(source).toContain("const handleConfigSync = useCallback")
    expect(source).toContain("handleConfigSync")
    expect(source).toMatch(/const handleConfigSync = useCallback\(\(value: string\) => \{\s*setConfiguration\(value\)\s*\}, \[\]\)/)
  })
})
