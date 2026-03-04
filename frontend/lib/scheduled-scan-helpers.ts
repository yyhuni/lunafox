export type ScheduledScanSelectionMode = "organization" | "target"

export interface ScheduledScanValidationInput {
  hasPreset: boolean
  currentStep: number
  name: string
  selectionMode: ScheduledScanSelectionMode
  selectedOrgId: number | null
  selectedTargetId: number | null
  selectedPresetId: string | null
  workflowIds: number[]
  configuration: string
  isYamlValid: boolean
  cronExpression: string
}

export type ScheduledScanValidationError =
  | "form.taskNameRequired"
  | "form.scanWorkflowRequired"
  | "form.configurationRequired"
  | "form.yamlInvalid"
  | "form.cronRequired"
  | "toast.selectOrganization"
  | "toast.selectTarget"

const isObjectRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null

export const isCronExpressionValid = (cronExpression: string): boolean => {
  const parts = cronExpression.trim().split(/\s+/)
  return parts.length === 5
}

export const validateScheduledScanStep = (
  input: ScheduledScanValidationInput
): ScheduledScanValidationError | null => {
  const {
    hasPreset,
    currentStep,
    name,
    selectionMode,
    selectedOrgId,
    selectedTargetId,
    selectedPresetId,
    workflowIds,
    configuration,
    isYamlValid,
    cronExpression,
  } = input

  if (hasPreset) {
    switch (currentStep) {
      case 1:
        return name.trim() ? null : "form.taskNameRequired"
      case 2:
        if (!selectedPresetId || workflowIds.length === 0) return "form.scanWorkflowRequired"
        return null
      case 3:
        if (!configuration.trim()) return "form.configurationRequired"
        if (!isYamlValid) return "form.yamlInvalid"
        return null
      case 4:
        return isCronExpressionValid(cronExpression) ? null : "form.cronRequired"
      default:
        return null
    }
  }

  switch (currentStep) {
    case 1:
      return name.trim() ? null : "form.taskNameRequired"
    case 2:
      if (selectionMode === "organization") {
        return selectedOrgId ? null : "toast.selectOrganization"
      }
      return selectedTargetId ? null : "toast.selectTarget"
    case 3:
      if (!selectedPresetId || workflowIds.length === 0) return "form.scanWorkflowRequired"
      return null
    case 4:
      if (!configuration.trim()) return "form.configurationRequired"
      if (!isYamlValid) return "form.yamlInvalid"
      return null
    case 5:
      return isCronExpressionValid(cronExpression) ? null : "form.cronRequired"
    default:
      return null
  }
}

export const getConfigConflictMessage = (error: unknown): string | null => {
  if (!isObjectRecord(error)) return null
  const response = error.response
  if (!isObjectRecord(response)) return null
  const data = response.data
  if (!isObjectRecord(data)) return null
  const detail = data.error
  if (!isObjectRecord(detail)) return null
  if (detail.code !== "CONFIG_CONFLICT") return null
  return typeof detail.message === "string" ? detail.message : ""
}
