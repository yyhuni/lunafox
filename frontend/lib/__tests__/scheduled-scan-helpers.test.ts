import { describe, expect, it } from "vitest"
import {
  getConfigConflictMessage,
  getNextCronExecutions,
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
  scanWorkflow: "subdomain_discovery",
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
  it("validates only parseable explicit five-field cron expressions", () => {
    expect(isCronExpressionValid("0 2 * * *")).toBe(true)
    expect(isCronExpressionValid("* * * * *")).toBe(true)
    expect(isCronExpressionValid("*/5 * * * *")).toBe(true)
    expect(isCronExpressionValid("61 2 * * *")).toBe(false)
    expect(isCronExpressionValid("0 0 2 * * *")).toBe(false)
    expect(isCronExpressionValid("@daily")).toBe(false)
    expect(isCronExpressionValid("CRON_TZ=UTC 0 2 * * *")).toBe(false)
    expect(isCronExpressionValid("")).toBe(false)
  })

  it("calculates previews in UTC", () => {
    expect(
      getNextCronExecutions("0 9 * * *", new Date("2026-01-01T00:00:00Z"), 1)
        .map((date) => date.toISOString())
    ).toEqual(["2026-01-01T09:00:00.000Z"])
  })

  it("validates preset flow steps", () => {
    expect(
      validateScheduledScanStep(withOverrides({ hasPreset: true, currentStep: 1, name: "   " }))
    ).toBe("form.taskNameRequired")
    expect(
      validateScheduledScanStep(withOverrides({ hasPreset: true, currentStep: 4, cronExpression: "0 2 * *" }))
    ).toBe("form.cronRequired")
    expect(
      validateScheduledScanStep(withOverrides({ hasPreset: true, currentStep: 2, scanWorkflow: null }))
    ).toBe("form.scanWorkflowRequired")
    expect(
      validateScheduledScanStep(withOverrides({ hasPreset: true, currentStep: 3, configuration: "" }))
    ).toBe("form.configurationRequired")
    expect(
      validateScheduledScanStep(withOverrides({ hasPreset: true, currentStep: 3, isYamlValid: false }))
    ).toBe("form.yamlInvalid")
  })

  it("validates full flow steps", () => {
    expect(
      validateScheduledScanStep(withOverrides({ currentStep: 1, name: "   " }))
    ).toBe("form.taskNameRequired")
    expect(
      validateScheduledScanStep(withOverrides({ currentStep: 1, selectionMode: "organization", selectedOrgId: null }))
    ).toBe("toast.selectOrganization")
    expect(
      validateScheduledScanStep(withOverrides({ currentStep: 1, selectionMode: "target", selectedTargetId: null }))
    ).toBe("toast.selectTarget")
    expect(
      validateScheduledScanStep(withOverrides({ currentStep: 4, cronExpression: "0 2 * *" }))
    ).toBe("form.cronRequired")
    expect(
      validateScheduledScanStep(withOverrides({ currentStep: 2, scanWorkflow: null }))
    ).toBe("form.scanWorkflowRequired")
    expect(
      validateScheduledScanStep(withOverrides({ currentStep: 3, configuration: "" }))
    ).toBe("form.configurationRequired")
    expect(
      validateScheduledScanStep(withOverrides({ currentStep: 3, isYamlValid: false }))
    ).toBe("form.yamlInvalid")
  })

  it("accepts complete four-step full and preset flows", () => {
    expect(validateScheduledScanStep(withOverrides({ currentStep: 1 }))).toBe(null)
    expect(validateScheduledScanStep(withOverrides({ currentStep: 2 }))).toBe(null)
    expect(validateScheduledScanStep(withOverrides({ currentStep: 3 }))).toBe(null)
    expect(validateScheduledScanStep(withOverrides({ currentStep: 4 }))).toBe(null)
    expect(validateScheduledScanStep(withOverrides({ hasPreset: true, currentStep: 1 }))).toBe(null)
    expect(validateScheduledScanStep(withOverrides({ hasPreset: true, currentStep: 2 }))).toBe(null)
    expect(validateScheduledScanStep(withOverrides({ hasPreset: true, currentStep: 3 }))).toBe(null)
    expect(validateScheduledScanStep(withOverrides({ hasPreset: true, currentStep: 4 }))).toBe(null)
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
