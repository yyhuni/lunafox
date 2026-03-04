import { describe, expect, it } from "vitest"
import {
  getApiErrorMessage,
  getInitiateScanValidationIssue,
  type InitiateScanValidationInput,
} from "@/lib/initiate-scan-helpers"

const baseInput: InitiateScanValidationInput = {
  selectMode: "preset",
  selectedPresetId: "preset-1",
  selectedWorkflowIds: [1],
  configuration: "config: true",
  isYamlValid: true,
  organizationId: 1,
}

const withOverrides = (
  overrides: Partial<InitiateScanValidationInput>
): InitiateScanValidationInput => ({
  ...baseInput,
  ...overrides,
})

describe("initiate scan helpers", () => {
  it("returns validation issues for preset mode", () => {
    expect(
      getInitiateScanValidationIssue(withOverrides({ selectedPresetId: null }))
    ).toEqual({ titleKey: "noPresetSelected" })
  })

  it("returns validation issues for custom mode", () => {
    expect(
      getInitiateScanValidationIssue(withOverrides({ selectMode: "custom", selectedWorkflowIds: [] }))
    ).toEqual({ titleKey: "noWorkflowSelected" })
  })

  it("validates configuration and yaml state", () => {
    expect(
      getInitiateScanValidationIssue(withOverrides({ configuration: "   " }))
    ).toEqual({ titleKey: "emptyConfig" })
    expect(
      getInitiateScanValidationIssue(withOverrides({ isYamlValid: false }))
    ).toEqual({ titleKey: "invalidConfig" })
  })

  it("requires organization or target id", () => {
    expect(
      getInitiateScanValidationIssue(withOverrides({ organizationId: undefined, targetId: undefined }))
    ).toEqual({ titleKey: "paramError", descriptionKey: "paramErrorDesc" })
  })

  it("returns null when inputs are valid", () => {
    expect(getInitiateScanValidationIssue(baseInput)).toBeNull()
  })

  it("extracts api error messages safely", () => {
    expect(
      getApiErrorMessage({ response: { data: { error: { message: "backend error" } } } })
    ).toBe("backend error")
    expect(getApiErrorMessage(new Error("boom"))).toBe("boom")
    expect(getApiErrorMessage({})).toBeNull()
  })
})
