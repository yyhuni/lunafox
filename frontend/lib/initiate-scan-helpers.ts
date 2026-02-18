export type InitiateScanSelectMode = "preset" | "custom"

export interface InitiateScanValidationInput {
  selectMode: InitiateScanSelectMode
  selectedPresetId: string | null
  selectedEngineIds: number[]
  configuration: string
  isYamlValid: boolean
  organizationId?: number
  targetId?: number
}

export interface InitiateScanValidationIssue {
  titleKey: "noPresetSelected" | "noEngineSelected" | "emptyConfig" | "invalidConfig" | "paramError"
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
    selectedEngineIds,
    configuration,
    isYamlValid,
    organizationId,
    targetId,
  } = input

  if (selectMode === "preset") {
    if (!selectedPresetId) return { titleKey: "noPresetSelected" }
  } else if (selectedEngineIds.length === 0) {
    return { titleKey: "noEngineSelected" }
  }

  if (!configuration.trim()) return { titleKey: "emptyConfig" }
  if (!isYamlValid) return { titleKey: "invalidConfig" }

  if (!organizationId && !targetId) {
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
