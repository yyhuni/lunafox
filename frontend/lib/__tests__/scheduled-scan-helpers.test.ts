import { describe, expect, it } from "vitest"
import {
  getConfigConflictMessage,
  isCronExpressionValid,
  validateScheduledScanStep,
  type ScheduledScanValidationInput,
} from "@/lib/scheduled-scan-helpers"

const baseInput: ScheduledScanValidationInput = {
  hasPreset: false,
  currentStep: 1,
  name: "Weekly Scan",
  selectionMode: "organization",
  selectedOrgId: 10,
  selectedTargetId: 20,
  selectedPresetId: "preset-1",
  workflowIds: [1],
  configuration: "scan: true",
  isYamlValid: true,
  cronExpression: "0 2 * * *",
}

const withOverrides = (
  overrides: Partial<ScheduledScanValidationInput>
): ScheduledScanValidationInput => ({
  ...baseInput,
  ...overrides,
})

describe("scheduled scan helpers", () => {
  it("validates cron expressions by segment count", () => {
    expect(isCronExpressionValid("0 2 * * *")).toBe(true)
    expect(isCronExpressionValid("*/5 * * *")).toBe(false)
    expect(isCronExpressionValid("")).toBe(false)
  })

  it("validates preset flow steps", () => {
    expect(
      validateScheduledScanStep(withOverrides({ hasPreset: true, currentStep: 1, name: "   " }))
    ).toBe("form.taskNameRequired")
    expect(
      validateScheduledScanStep(withOverrides({ hasPreset: true, currentStep: 2, selectedPresetId: null }))
    ).toBe("form.scanWorkflowRequired")
    expect(
      validateScheduledScanStep(withOverrides({ hasPreset: true, currentStep: 3, configuration: "" }))
    ).toBe("form.configurationRequired")
    expect(
      validateScheduledScanStep(withOverrides({ hasPreset: true, currentStep: 3, isYamlValid: false }))
    ).toBe("form.yamlInvalid")
    expect(
      validateScheduledScanStep(withOverrides({ hasPreset: true, currentStep: 4, cronExpression: "0 2 * *" }))
    ).toBe("form.cronRequired")
  })

  it("validates full flow steps", () => {
    expect(
      validateScheduledScanStep(withOverrides({ currentStep: 2, selectionMode: "organization", selectedOrgId: null }))
    ).toBe("toast.selectOrganization")
    expect(
      validateScheduledScanStep(withOverrides({ currentStep: 2, selectionMode: "target", selectedTargetId: null }))
    ).toBe("toast.selectTarget")
    expect(
      validateScheduledScanStep(withOverrides({ currentStep: 3, selectedPresetId: null }))
    ).toBe("form.scanWorkflowRequired")
    expect(
      validateScheduledScanStep(withOverrides({ currentStep: 4, configuration: "" }))
    ).toBe("form.configurationRequired")
    expect(
      validateScheduledScanStep(withOverrides({ currentStep: 5, cronExpression: "0 2 * *" }))
    ).toBe("form.cronRequired")
  })

  it("extracts config conflict messages safely", () => {
    expect(
      getConfigConflictMessage({
        response: { data: { error: { code: "CONFIG_CONFLICT", message: "duplicate" } } },
      })
    ).toBe("duplicate")
    expect(
      getConfigConflictMessage({
        response: { data: { error: { code: "CONFIG_CONFLICT" } } },
      })
    ).toBe("")
    expect(getConfigConflictMessage({})).toBe(null)
    expect(getConfigConflictMessage(null)).toBe(null)
  })
})
