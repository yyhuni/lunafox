import React from "react"
import { CronExpressionParser } from "cron-parser"
import cronstrue from "cronstrue/i18n"
import { toast } from "sonner"
import { useStep } from "@/hooks/use-step"
import { useCreateScheduledScan } from "@/hooks/use-scheduled-scans"
import { useEngines } from "@/hooks/use-engines"
import {
  getConfigConflictMessage,
  validateScheduledScanStep,
  type ScheduledScanSelectionMode,
} from "@/lib/scheduled-scan-helpers"
import type { CreateScheduledScanRequest } from "@/types/scheduled-scan.types"
import {
  useScheduledScanConfigState,
  useScheduledScanSearch,
} from "@/components/scan/scheduled/scheduled-scan-dialog-state-hooks"

type UseScheduledScanDialogStateProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
  presetOrganizationId?: number
  presetOrganizationName?: string
  presetTargetId?: number
  presetTargetName?: string
  hasPreset: boolean
  totalSteps: number
  locale: string
  t: (key: string, params?: Record<string, string | number | Date>) => string
}

export function useScheduledScanDialogState({
  open,
  onOpenChange,
  onSuccess,
  presetOrganizationId,
  presetOrganizationName,
  presetTargetId,
  presetTargetName,
  hasPreset,
  totalSteps,
  locale,
  t,
}: UseScheduledScanDialogStateProps) {
  const { mutate: createScheduledScan, isPending } = useCreateScheduledScan()
  const { data: enginesData } = useEngines()

  const {
    orgSearchInput,
    setOrgSearchInput,
    targetSearchInput,
    setTargetSearchInput,
    handleOrgSearch,
    handleTargetSearch,
    isOrgFetching,
    isTargetFetching,
    organizations,
    targets,
  } = useScheduledScanSearch({ open })

  const [currentStep, { goToNextStep, goToPrevStep, reset: resetStep }] = useStep(totalSteps)

  const [name, setName] = React.useState("")
  const [engineIds, setEngineIds] = React.useState<number[]>([])
  const [selectedPresetId, setSelectedPresetId] = React.useState<string | null>(null)
  const [selectionMode, setSelectionMode] = React.useState<ScheduledScanSelectionMode>("organization")
  const [selectedOrgId, setSelectedOrgId] = React.useState<number | null>(null)
  const [selectedTargetId, setSelectedTargetId] = React.useState<number | null>(null)
  const [cronExpression, setCronExpression] = React.useState("0 2 * * *")

  const {
    configuration,
    isConfigEdited,
    isYamlValid,
    showOverwriteConfirm,
    pendingConfigChange,
    setShowOverwriteConfirm,
    setPendingConfigChange,
    handlePresetConfigChange,
    handleManualConfigChange,
    handleOverwriteConfirm,
    handleOverwriteCancel,
    handleYamlValidationChange,
    resetConfigState,
  } = useScheduledScanConfigState()

  React.useEffect(() => {
    if (open) {
      if (presetOrganizationId) {
        setSelectionMode("organization")
        setSelectedOrgId(presetOrganizationId)
        setName(presetOrganizationName ? `${presetOrganizationName} - ${t("title")}` : "")
      } else if (presetTargetId) {
        setSelectionMode("target")
        setSelectedTargetId(presetTargetId)
        setName(presetTargetName ? `${presetTargetName} - ${t("title")}` : "")
      }
    }
  }, [open, presetOrganizationId, presetOrganizationName, presetTargetId, presetTargetName, t])

  const engines = React.useMemo(() => enginesData || [], [enginesData])

  const selectedEngines = React.useMemo(() => {
    if (!engineIds.length || !engines.length) return []
    return engines.filter((engine) => engineIds.includes(engine.id))
  }, [engineIds, engines])

  const resetForm = React.useCallback(() => {
    setName("")
    setEngineIds([])
    setSelectedPresetId(null)
    setSelectionMode("organization")
    setSelectedOrgId(null)
    setSelectedTargetId(null)
    setCronExpression("0 2 * * *")
    resetConfigState()
    resetStep()
  }, [resetConfigState, resetStep])

  const handleEngineIdsChange = React.useCallback((newEngineIds: number[]) => {
    setEngineIds(newEngineIds)
  }, [])

  const handleOpenChange = React.useCallback((isOpen: boolean) => {
    if (!isOpen) resetForm()
    onOpenChange(isOpen)
  }, [onOpenChange, resetForm])

  const handleOrgSelect = React.useCallback((orgId: number) => {
    setSelectedOrgId((prev) => (prev === orgId ? null : orgId))
  }, [])

  const handleTargetSelect = React.useCallback((targetId: number) => {
    setSelectedTargetId((prev) => (prev === targetId ? null : targetId))
  }, [])

  const validateCurrentStep = React.useCallback((): boolean => {
    const errorKey = validateScheduledScanStep({
      hasPreset,
      currentStep,
      name,
      selectionMode,
      selectedOrgId,
      selectedTargetId,
      selectedPresetId,
      engineIds,
      configuration,
      isYamlValid,
      cronExpression,
    })
    if (errorKey) {
      toast.error(t(errorKey))
      return false
    }
    return true
  }, [
    configuration,
    cronExpression,
    currentStep,
    engineIds,
    hasPreset,
    isYamlValid,
    name,
    selectedOrgId,
    selectedPresetId,
    selectedTargetId,
    selectionMode,
    t,
  ])

  const handleNext = React.useCallback(() => {
    if (validateCurrentStep()) {
      goToNextStep()
    }
  }, [goToNextStep, validateCurrentStep])

  const handleSubmit = React.useCallback(() => {
    if (!validateCurrentStep()) return
    const request: CreateScheduledScanRequest = {
      name: name.trim(),
      configuration: configuration.trim(),
      engineIds: engineIds,
      engineNames: selectedEngines.map((engine) => engine.name),
      cronExpression: cronExpression.trim(),
    }
    if (selectionMode === "organization" && selectedOrgId) {
      request.organizationId = selectedOrgId
    } else if (selectedTargetId) {
      request.targetId = selectedTargetId
    }

    createScheduledScan(request, {
      onSuccess: () => {
        resetForm()
        onOpenChange(false)
        onSuccess?.()
      },
      onError: (err: unknown) => {
        const conflictMessage = getConfigConflictMessage(err)
        if (conflictMessage !== null) {
          toast.error(t("toast.configConflict"), {
            description: conflictMessage,
          })
        }
      },
    })
  }, [
    configuration,
    createScheduledScan,
    cronExpression,
    engineIds,
    name,
    onOpenChange,
    onSuccess,
    resetForm,
    selectedEngines,
    selectedOrgId,
    selectedTargetId,
    selectionMode,
    t,
    validateCurrentStep,
  ])

  const getCronDescription = React.useCallback((cron: string): string => {
    try {
      const parts = cron.trim().split(/\s+/)
      if (parts.length !== 5) return t("form.invalidExpression")
      return cronstrue.toString(cron, { locale: locale === "zh" ? "zh_CN" : "en" })
    } catch {
      return t("form.invalidExpression")
    }
  }, [locale, t])

  const getNextExecutions = React.useCallback((cron: string, count: number = 3): string[] => {
    try {
      const parts = cron.trim().split(/\s+/)
      if (parts.length !== 5) return []
      const interval = CronExpressionParser.parse(cron, { currentDate: new Date(), tz: "Asia/Shanghai" })
      const results: string[] = []
      for (let i = 0; i < count; i++) {
        const next = interval.next()
        results.push(next.toDate().toLocaleString(locale === "zh" ? "zh-CN" : "en-US"))
      }
      return results
    } catch {
      return []
    }
  }, [locale])

  return {
    isPending,
    orgSearchInput,
    setOrgSearchInput,
    targetSearchInput,
    setTargetSearchInput,
    handleOrgSearch,
    handleTargetSearch,
    isOrgFetching,
    isTargetFetching,
    currentStep,
    goToPrevStep,
    name,
    setName,
    engineIds,
    setEngineIds,
    selectedPresetId,
    setSelectedPresetId,
    selectionMode,
    setSelectionMode,
    selectedOrgId,
    selectedTargetId,
    setSelectedOrgId,
    setSelectedTargetId,
    cronExpression,
    setCronExpression,
    configuration,
    isConfigEdited,
    isYamlValid,
    showOverwriteConfirm,
    pendingConfigChange,
    targets,
    engines,
    organizations,
    selectedEngines,
    handlePresetConfigChange,
    handleManualConfigChange,
    handleEngineIdsChange,
    handleOverwriteConfirm,
    handleOverwriteCancel,
    handleYamlValidationChange,
    handleOpenChange,
    handleOrgSelect,
    handleTargetSelect,
    handleNext,
    handleSubmit,
    getCronDescription,
    getNextExecutions,
    setShowOverwriteConfirm,
    setPendingConfigChange,
  }
}
