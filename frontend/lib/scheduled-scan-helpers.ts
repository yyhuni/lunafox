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
  timeZone: string
  cronExpression: string
}

export type ScheduledScanValidationError =
  | "form.taskNameRequired"
  | "form.scanWorkflowRequired"
  | "form.configurationRequired"
  | "form.noEnabledSteps"
  | "form.yamlInvalid"
  | "form.timeZoneRequired"
  | "form.timeZoneInvalid"
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

const fixedOffsetTimeZonePattern = /^(?:UTC|GMT)?[+-]\d{1,2}(?::?\d{2})?$/i

export const isIanaTimeZoneValid = (timeZone: string): boolean => {
  const zone = timeZone.trim()
  if (!zone || zone === "Local" || fixedOffsetTimeZonePattern.test(zone)) return false

  try {
    new Intl.DateTimeFormat("en-US", { timeZone: zone }).format()
    return true
  } catch {
    return false
  }
}

export const getBrowserTimeZone = (): string => {
  if (typeof Intl === "undefined") return ""
  const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone ?? ""
  return isIanaTimeZoneValid(timeZone) ? timeZone : ""
}

export const formatScheduledScanInstant = (
  dateString: string,
  locale: string,
  viewerTimeZone: string = getBrowserTimeZone()
): string => {
  const date = new Date(dateString)
  if (Number.isNaN(date.getTime())) return "-"

  return date.toLocaleString(locale, {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    ...(viewerTimeZone ? { timeZone: viewerTimeZone } : {}),
  })
}

export const getSupportedTimeZones = (selectedTimeZone?: string): string[] => {
  let supported: string[] = []
  try {
    supported = Intl.supportedValuesOf("timeZone")
  } catch {
    // Older browsers still let a valid selected zone be displayed and validated.
  }

  const zones = new Set(["UTC", ...supported])
  const selected = selectedTimeZone?.trim()
  if (selected && isIanaTimeZoneValid(selected)) zones.add(selected)

  return [...zones].sort((left, right) => {
    if (left === "UTC") return -1
    if (right === "UTC") return 1
    return left.localeCompare(right)
  })
}

export const getNextCronExecutions = (
  cronExpression: string,
  timeZone: string,
  currentDate: Date = new Date(),
  count: number = 3
): Date[] => {
  if (!isCronExpressionValid(cronExpression) || !isIanaTimeZoneValid(timeZone) || count <= 0) {
    return []
  }

  try {
    const interval = CronExpressionParser.parse(cronExpression.trim(), {
      currentDate,
      tz: timeZone.trim(),
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
    timeZone,
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
      if (!timeZone.trim()) return "form.timeZoneRequired"
      if (!isIanaTimeZoneValid(timeZone)) return "form.timeZoneInvalid"
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
