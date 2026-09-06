import { CronExpressionParser } from "cron-parser"
import { hasNoEnabledWorkflowSteps } from "@/lib/workflow-config"

export type ScheduledScanSelectionMode = "organization" | "target"

export interface ScheduledScanValidationInput {
  hasPreset: boolean
  currentStep: number
  name: string
  selectionMode: ScheduledScanSelectionMode
  selectedOrgId: number | null
  selectedTargetId: number | null
  scanWorkflow: string | null
  configuration: string
  isYamlValid: boolean
  cronExpression: string
}

export type ScheduledScanValidationError =
  | "form.taskNameRequired"
  | "form.scanWorkflowRequired"
  | "form.configurationRequired"
  | "form.noEnabledSteps"
  | "form.yamlInvalid"
  | "form.cronRequired"
  | "toast.selectOrganization"
  | "toast.selectTarget"

const isObjectRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null

export const isCronExpressionValid = (cronExpression: string): boolean => {
  const expression = cronExpression.trim()
  if (!expression || expression.startsWith("@")) return false

  const parts = expression.split(/\s+/)
  if (parts.length !== 5) return false
  if (parts.some((part) => /^(?:CRON_)?TZ=/i.test(part))) return false

  try {
    CronExpressionParser.parse(expression)
    return true
  } catch {
    return false
  }
}

export const getNextCronExecutions = (
  cronExpression: string,
  currentDate: Date = new Date(),
  count: number = 3
): Date[] => {
  if (!isCronExpressionValid(cronExpression) || count <= 0) {
    return []
  }

  try {
    const interval = CronExpressionParser.parse(cronExpression.trim(), {
      currentDate,
      tz: "UTC",
    })
    return Array.from({ length: count }, () => interval.next().toDate())
  } catch {
    return []
  }
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
    scanWorkflow,
    configuration,
    isYamlValid,
    cronExpression,
  } = input

  switch (currentStep) {
    case 1:
      if (!name.trim()) return "form.taskNameRequired"
      if (!hasPreset) {
        if (selectionMode === "organization" && !selectedOrgId) return "toast.selectOrganization"
        if (selectionMode === "target" && !selectedTargetId) return "toast.selectTarget"
      }
      return null
    case 2:
      return scanWorkflow ? null : "form.scanWorkflowRequired"
    case 3:
      if (!configuration.trim()) return "form.configurationRequired"
      if (!isYamlValid) return "form.yamlInvalid"
      if (hasNoEnabledWorkflowSteps(configuration)) return "form.noEnabledSteps"
      return null
    case 4:
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
