import { hasNoEnabledWorkflowSteps } from "@/lib/workflow-config"

export type InitiateScanSelectMode = "preset" | "custom"

export interface InitiateScanValidationInput {
  selectMode: InitiateScanSelectMode
  selectedPresetId: string | null
  selectedScanWorkflowName: string | null
  configuration: string
  isYamlValid: boolean
  organizationId?: number
  targetId?: number
  organizationIds?: number[]
  targetIds?: number[]
}

export interface InitiateScanValidationIssue {
  titleKey: "noPresetSelected" | "noWorkflowSelected" | "emptyConfig" | "invalidConfig" | "noEnabledSteps" | "paramError"
  descriptionKey?: "paramErrorDesc"
}

const isObjectRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null

export const getInitiateScanValidationIssue = (
  input: InitiateScanValidationInput
): InitiateScanValidationIssue | null => {
  const {
    selectMode,
    selectedPresetId,
    selectedScanWorkflowName,
    configuration,
    isYamlValid,
    organizationId,
    targetId,
    organizationIds,
    targetIds,
  } = input

  if (selectMode === "preset") {
    if (!selectedPresetId) return { titleKey: "noPresetSelected" }
  } else if (!selectedScanWorkflowName) {
    return { titleKey: "noWorkflowSelected" }
  }

  if (!configuration.trim()) return { titleKey: "emptyConfig" }
  if (!isYamlValid) return { titleKey: "invalidConfig" }

  if (hasNoEnabledWorkflowSteps(configuration)) return { titleKey: "noEnabledSteps" }

  const hasSingleScope = Boolean(organizationId || targetId)
  const hasBulkScope = Boolean(organizationIds?.length || targetIds?.length)

  if (!hasSingleScope && !hasBulkScope) {
    return { titleKey: "paramError", descriptionKey: "paramErrorDesc" }
  }

  return null
}

export const getApiErrorMessage = (error: unknown): string | null => {
  if (isObjectRecord(error)) {
    const response = error.response
    if (isObjectRecord(response)) {
      const data = response.data
      if (isObjectRecord(data)) {
        const detail = data.error
        if (isObjectRecord(detail) && typeof detail.message === "string") {
          return detail.message
        }
      }
    }
  }

  if (error instanceof Error) return error.message
  return null
}
