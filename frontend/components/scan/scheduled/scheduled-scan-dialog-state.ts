import React from "react"
import cronstrue from "cronstrue/i18n"
import { toastFeedback } from "@/lib/toast-helpers"
import { useStep } from "@/hooks/use-step"
import { useCreateScheduledScan } from "@/hooks/use-scheduled-scans"
import { useLoadScanWorkflowProfile, useScanWorkflows } from "@/hooks/use-scan-workflows"
import { useEngineCatalogDetails } from "@/hooks/use-engine-catalog"
import {
  getConfigConflictMessage,
  getNextCronExecutions,
  validateScheduledScanStep,
  type ScheduledScanSelectionMode,
} from "@/lib/scheduled-scan-helpers"
import {
  adaptWorkflowProfile,
  hasNoEnabledWorkflowSteps,
  serializeCanonicalWorkflowConfiguration,
  serializeWorkflowProfileDraft,
} from "@/lib/workflow-config"
import { buildWorkflowWithEngineCatalog } from "@/lib/engine-catalog"
import type { Locale } from "@/i18n/config"
import type { CreateScheduledScanRequest } from "@/types/scheduled-scan.types"
import type { ScanInputSource } from "@/types/scan.types"
import type { ScanConfigValidationHandle } from "@/components/scan/scan-config-view-toggle"
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
  locale: Locale
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
  const loadScanWorkflowProfile = useLoadScanWorkflowProfile()
  const {
    data: workflowsData,
    isLoading: isLoadingWorkflows,
    isError: isWorkflowsError,
  } = useScanWorkflows()

  const {
    orgSearchInput,
    setOrgSearchInput,
    orgPageSize,
    setOrgPageSize,
    organizationTotalCount,
    organizationPaginationNavigation,
    onFirstOrgPage,
    onPreviousOrgPage,
    onNextOrgPage,
    targetSearchInput,
    setTargetSearchInput,
    targetPageSize,
    setTargetPageSize,
    targetTotalCount,
    targetPaginationNavigation,
    onFirstTargetPage,
    onPreviousTargetPage,
    onNextTargetPage,
    handleOrgSearch,
    isOrgFetching,
    isTargetFetching,
    organizations,
    targets,
  } = useScheduledScanSearch({ open })

  const [currentStep, { goToNextStep, goToPrevStep, reset: resetStep }] = useStep(totalSteps)

  const [name, setName] = React.useState("")
  const [selectedScanWorkflowName, setSelectedScanWorkflowName] = React.useState<string | null>(null)
  const [selectionMode, setSelectionMode] = React.useState<ScheduledScanSelectionMode>("organization")
  const [selectedOrgId, setSelectedOrgId] = React.useState<number | null>(null)
  const [selectedTargetId, setSelectedTargetId] = React.useState<number | null>(null)
  const [selectedAgentID, setSelectedAgentID] = React.useState<number | null>(null)
  const [inputSource, setInputSource] = React.useState<ScanInputSource>("scanSnapshot")
  const [cronExpression, setCronExpression] = React.useState("0 2 * * *")
  const [isWorkflowConfigLoading, setIsWorkflowConfigLoading] = React.useState(false)
  const [pendingWorkflowName, setPendingWorkflowName] = React.useState<string | null>(null)
  const configValidationRef = React.useRef<ScanConfigValidationHandle | null>(null)
  const workflowSelectionRequestRef = React.useRef(0)

  const {
    configuration,
    isConfigEdited,
    isYamlValid,
    showOverwriteConfirm,
    pendingConfigChange,
    setShowOverwriteConfirm,
    setPendingConfigChange,
    handlePresetConfigChange,
    applyProfileConfiguration,
    handleConfigSync,
    handleManualConfigChange,
    handleOverwriteConfirm: handleConfigOverwriteConfirm,
    handleOverwriteCancel: handleConfigOverwriteCancel,
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

  const workflows = React.useMemo(() => workflowsData || [], [workflowsData])

  const selectedScanWorkflow = React.useMemo(() => {
    if (!selectedScanWorkflowName || !workflows.length) return null
    return workflows.find((item) => item.name === selectedScanWorkflowName) || null
  }, [selectedScanWorkflowName, workflows])
  const selectedEngineIds = React.useMemo(
    () => selectedScanWorkflow?.steps?.map((step) => step.engineId) ?? [],
    [selectedScanWorkflow]
  )
  const engineCatalog = useEngineCatalogDetails(selectedEngineIds)

  const selectedWorkflowWithEngines = React.useMemo(() => {
    if (!selectedScanWorkflow || !engineCatalog.data) return undefined
    try {
      return buildWorkflowWithEngineCatalog(selectedScanWorkflow, engineCatalog.data, locale)
    } catch {
      return undefined
    }
  }, [engineCatalog.data, locale, selectedScanWorkflow])

  const resetForm = React.useCallback(() => {
    workflowSelectionRequestRef.current += 1
    setName("")
    setSelectedScanWorkflowName(null)
    setSelectionMode("organization")
    setSelectedOrgId(null)
    setSelectedTargetId(null)
    setSelectedAgentID(null)
    setInputSource("scanSnapshot")
    setCronExpression("0 2 * * *")
    setIsWorkflowConfigLoading(false)
    setPendingWorkflowName(null)
    resetConfigState()
    resetStep()
  }, [resetConfigState, resetStep])

  const loadProfileConfiguration = React.useCallback(async (workflowName: string) => {
    const workflow = workflows.find((item) => item.name === workflowName)
    if (!workflow) throw new Error("Selected Workflow is unavailable")

    const profile = await loadScanWorkflowProfile(workflowName)
    return serializeWorkflowProfileDraft(adaptWorkflowProfile(profile, workflow))
  }, [loadScanWorkflowProfile, workflows])

  const applyWorkflowSelection = React.useCallback((workflowName: string, nextConfiguration: string) => {
    setSelectedScanWorkflowName(workflowName)
    applyProfileConfiguration(nextConfiguration)
    setIsWorkflowConfigLoading(false)
  }, [applyProfileConfiguration])

  const handleWorkflowNamesChange = React.useCallback(async (workflowNames: string[]) => {
    const nextWorkflowName = workflowNames.at(-1) ?? null
    const requestId = workflowSelectionRequestRef.current + 1
    workflowSelectionRequestRef.current = requestId

    if (!nextWorkflowName) {
      setPendingWorkflowName(null)
      setSelectedScanWorkflowName(null)
      applyProfileConfiguration("")
      setIsWorkflowConfigLoading(false)
      return
    }

    setIsWorkflowConfigLoading(true)
    try {
      const nextConfiguration = await loadProfileConfiguration(nextWorkflowName)
      if (workflowSelectionRequestRef.current !== requestId) return

      if (isConfigEdited && configuration !== nextConfiguration) {
        setPendingWorkflowName(nextWorkflowName)
        handlePresetConfigChange(nextConfiguration)
        setIsWorkflowConfigLoading(false)
        return
      }

      applyWorkflowSelection(nextWorkflowName, nextConfiguration)
    } catch {
      if (workflowSelectionRequestRef.current !== requestId) return
      setIsWorkflowConfigLoading(false)
      if (!isConfigEdited) {
        setSelectedScanWorkflowName(null)
        applyProfileConfiguration("")
      }
    }
  }, [applyProfileConfiguration, applyWorkflowSelection, configuration, handlePresetConfigChange, isConfigEdited, loadProfileConfiguration])

  const handleResetWorkflowConfig = React.useCallback(async () => {
    if (!selectedScanWorkflowName) return

    setIsWorkflowConfigLoading(true)
    try {
      applyProfileConfiguration(await loadProfileConfiguration(selectedScanWorkflowName))
    } finally {
      setIsWorkflowConfigLoading(false)
    }
  }, [applyProfileConfiguration, loadProfileConfiguration, selectedScanWorkflowName])

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
    if (currentStep === 3 && hasNoEnabledWorkflowSteps(configuration)) {
      toastFeedback.error(t("form.noEnabledSteps"))
      return false
    }
    if (currentStep === 2 && isWorkflowConfigLoading) {
      toastFeedback.error(t("form.configurationRequired"))
      return false
    }
    if (currentStep === 3 && (engineCatalog.isLoading || engineCatalog.isError || !engineCatalog.data)) {
      toastFeedback.error(t("toast.configConflict"))
      return false
    }
    const errorKey = validateScheduledScanStep({
      hasPreset,
      currentStep,
      name,
      selectionMode,
      selectedOrgId,
      selectedTargetId,
      scanWorkflow: selectedScanWorkflow?.name ?? null,
      configuration,
      isYamlValid,
      cronExpression,
    })
    if (errorKey) {
      toastFeedback.error(t(errorKey))
      return false
    }
    return true
  }, [
    configuration,
    cronExpression,
    currentStep,
    engineCatalog.data,
    engineCatalog.isError,
    engineCatalog.isLoading,
    hasPreset,
    isWorkflowConfigLoading,
    isYamlValid,
    name,
    selectedOrgId,
    selectedTargetId,
    selectedScanWorkflow,
    selectionMode,
    t,
  ])

  const handleNext = React.useCallback(() => {
    if (!validateCurrentStep()) return
    // The editor unmounts on step 4, so reveal resource errors while it can still focus them.
    if (currentStep === 3 && !configValidationRef.current?.validateAndReveal()) return
    goToNextStep()
  }, [currentStep, goToNextStep, validateCurrentStep])

  const handleSubmit = React.useCallback(() => {
    if (!validateCurrentStep()) return
    if (!selectedScanWorkflow) {
      toastFeedback.error(t("form.scanWorkflowRequired"))
      return
    }
    let canonicalConfiguration: ReturnType<typeof serializeCanonicalWorkflowConfiguration>
    try {
      if (!selectedWorkflowWithEngines) {
        throw new Error("Selected Workflow Engine catalog is unavailable")
      }
      canonicalConfiguration = serializeCanonicalWorkflowConfiguration(
        configuration,
        selectedScanWorkflow,
        selectedWorkflowWithEngines,
      )
    } catch (error) {
      toastFeedback.error(t("form.yamlInvalid"), {
        description: error instanceof Error ? error.message : undefined,
      })
      return
    }
    const request: CreateScheduledScanRequest = {
      displayName: name.trim(),
      configuration: canonicalConfiguration,
      scanWorkflow: selectedScanWorkflow.name,
      inputSource,
      cronExpression: cronExpression.trim(),
    }
    if (selectionMode === "organization" && selectedOrgId) {
      request.organizationId = selectedOrgId
    } else if (selectedTargetId) {
      request.targetId = selectedTargetId
    }
    if (selectedAgentID !== null) {
      request.agentId = selectedAgentID
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
          toastFeedback.error(t("toast.configConflict"), {
            description: conflictMessage,
          })
        }
      },
    })
  }, [
    configuration,
    createScheduledScan,
    cronExpression,
    name,
    onOpenChange,
    onSuccess,
    resetForm,
    selectedOrgId,
    selectedTargetId,
    selectedAgentID,
    inputSource,
    selectedScanWorkflow,
    selectedWorkflowWithEngines,
    selectionMode,
    t,
    validateCurrentStep,
  ])

  const handleOverwriteConfirm = React.useCallback(() => {
    handleConfigOverwriteConfirm()
    if (pendingWorkflowName !== null) {
      setSelectedScanWorkflowName(pendingWorkflowName)
      setPendingWorkflowName(null)
    }
  }, [handleConfigOverwriteConfirm, pendingWorkflowName])

  const handleOverwriteCancel = React.useCallback(() => {
    handleConfigOverwriteCancel()
    setPendingWorkflowName(null)
  }, [handleConfigOverwriteCancel])

  const getCronDescription = React.useCallback((cron: string): string => {
    try {
      const parts = cron.trim().split(/\s+/)
      if (parts.length !== 5) return t("form.invalidExpression")
      return cronstrue.toString(cron, { locale: locale === "zh" ? "zh_CN" : "en" })
    } catch {
      return t("form.invalidExpression")
    }
  }, [locale, t])

  const getNextExecutions = React.useCallback((cron: string, count: number = 3): string[] => (
    getNextCronExecutions(cron, new Date(), count).map((next) => (
      next.toLocaleString(locale === "zh" ? "zh-CN" : "en-US", { timeZone: "UTC" })
    ))
  ), [locale])

  return {
    isPending,
    configValidationRef,
    orgSearchInput,
    setOrgSearchInput,
    orgPageSize,
    setOrgPageSize,
    organizationTotalCount,
    organizationPaginationNavigation,
    onFirstOrgPage,
    onPreviousOrgPage,
    onNextOrgPage,
    targetSearchInput,
    setTargetSearchInput,
    targetPageSize,
    setTargetPageSize,
    targetTotalCount,
    targetPaginationNavigation,
    onFirstTargetPage,
    onPreviousTargetPage,
    onNextTargetPage,
    handleOrgSearch,
    isOrgFetching,
    isTargetFetching,
    currentStep,
    goToPrevStep,
    name,
    setName,
    selectedScanWorkflowName,
    isLoadingWorkflows,
    isWorkflowsError,
    isWorkflowConfigLoading,
    selectionMode,
    setSelectionMode,
    selectedOrgId,
    selectedTargetId,
    selectedAgentID,
    inputSource,
    setSelectedOrgId,
    setSelectedTargetId,
    setSelectedAgentID,
    setInputSource,
    cronExpression,
    setCronExpression,
    configuration,
    isConfigEdited,
    isYamlValid,
    showOverwriteConfirm,
    pendingConfigChange,
    targets,
    workflows,
    organizations,
    selectedScanWorkflows: selectedScanWorkflow ? [selectedScanWorkflow] : [],
    engineCatalogDetails: engineCatalog.data,
    isEngineCatalogLoading: engineCatalog.isLoading,
    isEngineCatalogError: engineCatalog.isError,
    handlePresetConfigChange,
    handleConfigSync,
    loadScanWorkflowProfile,
    handleManualConfigChange,
    handleWorkflowNamesChange,
    handleResetWorkflowConfig,
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
