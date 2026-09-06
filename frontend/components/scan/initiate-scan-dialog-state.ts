import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { toastFeedback } from "@/lib/toast-helpers"
import {
  adaptWorkflowProfile,
  hasNoEnabledWorkflowSteps,
  serializeCanonicalWorkflowConfiguration,
  serializeWorkflowProfileDraft,
} from "@/lib/workflow-config"
import { hasApiErrorReason } from "@/lib/api-error-info"
import { getInitiateScanValidationIssue } from "@/lib/initiate-scan-helpers"
import { useBulkInitiateScan, useInitiateScan } from "@/hooks/use-scans"
import { useLoadScanWorkflowProfile, useScanWorkflows } from "@/hooks/use-scan-workflows"
import { useEngineCatalogDetails } from "@/hooks/use-engine-catalog"
import { buildWorkflowWithEngineCatalog } from "@/lib/engine-catalog"
import type { Locale } from "@/i18n/config"
import type { ScanConfigValidationHandle } from "@/components/scan/scan-config-view-toggle"
import type { ScanInputSource } from "@/types/scan.types"

type UseInitiateScanDialogStateProps = {
  open?: boolean
  organizationId?: number
  targetId?: number
  organizationIds?: number[]
  targetIds?: number[]
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
  locale?: Locale
  tToast: (key: string, params?: Record<string, string | number | Date>) => string
}

export function useInitiateScanDialogState({
  open = false,
  organizationId,
  targetId,
  organizationIds,
  targetIds,
  onOpenChange,
  onSuccess,
  locale = "en",
  tToast,
}: UseInitiateScanDialogStateProps) {
  const initiateScanMutation = useInitiateScan()
  const bulkInitiateScanMutation = useBulkInitiateScan()
  const loadScanWorkflowProfile = useLoadScanWorkflowProfile()
  const [selectedWorkflowNames, setSelectedWorkflowNames] = useState<string[]>([])
  const [selectedAgentID, setSelectedAgentID] = useState<number | null>(null)
  const [inputSource, setInputSource] = useState<ScanInputSource>("scanSnapshot")
  const selectMode = "custom" as const
  const [currentStep, setCurrentStep] = useState(1)

  const [configuration, setConfiguration] = useState("")
  const [isConfigEdited, setIsConfigEdited] = useState(false)
  const [isYamlValid, setIsYamlValid] = useState(true)
  const [showOverwriteConfirm, setShowOverwriteConfirm] = useState(false)
  const [pendingConfigChange, setPendingConfigChange] = useState<string | null>(null)
  const [pendingWorkflowNames, setPendingWorkflowNames] = useState<string[] | null>(null)
  const [isWorkflowConfigLoading, setIsWorkflowConfigLoading] = useState(false)
  const [requiresProfileReview, setRequiresProfileReview] = useState(false)
  const configValidationRef = useRef<ScanConfigValidationHandle | null>(null)
  // Workflow details load after selection; stale responses must not replace a newer user choice.
  const workflowSelectionRequestRef = useRef(0)

  const { data: workflows, isLoading: isLoadingWorkflows, isError: isWorkflowsError } = useScanWorkflows()

  const selectedWorkflows = useMemo(() => {
    if (!selectedWorkflowNames.length || !workflows) return []
    const selectedSet = new Set(selectedWorkflowNames)
    return workflows.filter((item) => selectedSet.has(item.name))
  }, [selectedWorkflowNames, workflows])
  const selectedEngineIds = useMemo(
    () => selectedWorkflows.flatMap((workflow) => workflow.steps?.map((step) => step.engineId) ?? []),
    [selectedWorkflows]
  )
  const engineCatalog = useEngineCatalogDetails(selectedEngineIds)

  const selectedWorkflowWithEngines = useMemo(() => {
    const workflow = selectedWorkflows[0]
    if (!workflow || !engineCatalog.data) return undefined
    try {
      return buildWorkflowWithEngineCatalog(workflow, engineCatalog.data, locale)
    } catch {
      return undefined
    }
  }, [engineCatalog.data, locale, selectedWorkflows])

  const handleManualConfigChange = useCallback((value: string) => {
    setConfiguration(value)
    setIsConfigEdited(true)
  }, [])

  const handleConfigSync = useCallback((value: string) => {
    setConfiguration(value)
  }, [])

  const loadProfileConfiguration = useCallback(async (workflowName: string) => {
    const profile = await loadScanWorkflowProfile(workflowName)
    const workflow = workflows?.find((item) => item.name === workflowName)
    if (!workflow) throw new Error("Selected Workflow is unavailable")
    const draft = adaptWorkflowProfile(profile, workflow)
    return serializeWorkflowProfileDraft(draft)
  }, [loadScanWorkflowProfile, workflows])

  const applyWorkflowSelection = useCallback((workflowNames: string[], nextConfig: string) => {
    setSelectedWorkflowNames(workflowNames)
    setConfiguration(nextConfig)
    setIsConfigEdited(false)
    setIsYamlValid(true)
    setIsWorkflowConfigLoading(false)
    setRequiresProfileReview(false)
  }, [])

  const handleResetWorkflowConfig = useCallback(async () => {
    const workflowName = selectedWorkflowNames[0]
    if (!workflowName) return
    const nextConfig = await loadProfileConfiguration(workflowName)
    setConfiguration(nextConfig)
    setIsConfigEdited(false)
    setIsYamlValid(true)
    setRequiresProfileReview(false)
  }, [loadProfileConfiguration, selectedWorkflowNames])

  const handleWorkflowNamesChange = useCallback(async (workflowNames: string[]) => {
    const nextWorkflowNames = workflowNames.length > 0 ? [workflowNames[workflowNames.length - 1]] : []
    const requestId = workflowSelectionRequestRef.current + 1
    workflowSelectionRequestRef.current = requestId

    if (!isConfigEdited) {
      setSelectedWorkflowNames(nextWorkflowNames)
      setIsYamlValid(true)
      setIsWorkflowConfigLoading(nextWorkflowNames.length > 0)
      if (nextWorkflowNames.length === 0) {
        setConfiguration("")
        setIsConfigEdited(false)
        setIsWorkflowConfigLoading(false)
        return
      }
    }

    let nextConfig = ""
    try {
      nextConfig = nextWorkflowNames.length > 0 ? await loadProfileConfiguration(nextWorkflowNames[0]) : ""
    } catch {
      if (workflowSelectionRequestRef.current === requestId) {
        setIsWorkflowConfigLoading(false)
        if (!isConfigEdited) {
          setSelectedWorkflowNames([])
          setConfiguration("")
        }
      }
      return
    }
    if (workflowSelectionRequestRef.current !== requestId) return

    if (isConfigEdited && configuration !== nextConfig) {
      setPendingWorkflowNames(nextWorkflowNames)
      setPendingConfigChange(nextConfig)
      setShowOverwriteConfirm(true)
      setIsWorkflowConfigLoading(false)
      return
    }
    applyWorkflowSelection(nextWorkflowNames, nextConfig)
  }, [applyWorkflowSelection, configuration, isConfigEdited, loadProfileConfiguration])

  useEffect(() => {
    const firstWorkflow = workflows?.find((workflow) => workflow.isExecutable)
    if (
      !open ||
      isLoadingWorkflows ||
      isWorkflowsError ||
      selectedWorkflowNames.length > 0 ||
      !firstWorkflow?.name
    ) {
      return
    }

    void handleWorkflowNamesChange([firstWorkflow.name])
  }, [handleWorkflowNamesChange, isLoadingWorkflows, isWorkflowsError, open, selectedWorkflowNames.length, workflows])

  const handleOverwriteConfirm = useCallback(() => {
    if (pendingConfigChange !== null) {
      const nextWorkflowNames = pendingWorkflowNames ?? selectedWorkflowNames
      applyWorkflowSelection(nextWorkflowNames, pendingConfigChange)
    }
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
    setPendingWorkflowNames(null)
  }, [applyWorkflowSelection, pendingConfigChange, pendingWorkflowNames, selectedWorkflowNames])

  const handleOverwriteCancel = useCallback(() => {
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
    setPendingWorkflowNames(null)
  }, [])

  const handleYamlValidationChange = useCallback((isValid: boolean) => {
    setIsYamlValid(isValid)
  }, [])

  const resetDialogState = useCallback(() => {
    workflowSelectionRequestRef.current += 1
    setSelectedWorkflowNames([])
    setSelectedAgentID(null)
    setInputSource("scanSnapshot")
    setConfiguration("")
    setIsConfigEdited(false)
    setIsYamlValid(true)
    setCurrentStep(1)
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
    setPendingWorkflowNames(null)
    setIsWorkflowConfigLoading(false)
    setRequiresProfileReview(false)
  }, [])

  const handleInitiate = useCallback(async () => {
    const issue = getInitiateScanValidationIssue({
      selectMode,
      selectedPresetId: null,
      selectedScanWorkflowName: selectedWorkflowNames[0] ?? null,
      configuration,
      isYamlValid,
      organizationId,
      targetId,
      organizationIds,
      targetIds,
    })
    if (issue) {
      toastFeedback.error(
        tToast(issue.titleKey),
        issue.descriptionKey ? { description: tToast(issue.descriptionKey) } : undefined
      )
      return
    }
    if (!configValidationRef.current?.validateAndReveal()) {
      return
    }

    const scanWorkflow = selectedWorkflowNames[0]
    if (!scanWorkflow) {
      toastFeedback.error(tToast("noWorkflowSelected"))
      return
    }

    try {
      const selectedWorkflow = selectedWorkflows[0]
      if (!selectedWorkflow) {
        toastFeedback.error(tToast("noWorkflowSelected"))
        return
      }
      if (!selectedWorkflowWithEngines) {
        throw new Error("Selected Workflow Engine catalog is unavailable")
      }
      const parsedConfiguration = serializeCanonicalWorkflowConfiguration(
        configuration,
        selectedWorkflow,
        selectedWorkflowWithEngines,
      )
      const hasBulkScope = Boolean(organizationIds?.length || targetIds?.length)

      if (hasBulkScope) {
        await bulkInitiateScanMutation.mutateAsync({
          organizationIds,
          targetIds,
          configuration: parsedConfiguration,
          scanWorkflow,
          agentId: selectedAgentID ?? undefined,
          inputSource,
        })
      } else {
        await initiateScanMutation.mutateAsync({
          organizationId,
          targetId,
          configuration: parsedConfiguration,
          scanWorkflow,
          agentId: selectedAgentID ?? undefined,
          inputSource,
        })
      }
      onSuccess?.()
      onOpenChange(false)
      resetDialogState()
    } catch (error) {
      if (error instanceof Error && error.name === "WorkflowConfigurationDraftError") {
        toastFeedback.error(tToast("invalidConfig"), { description: error.message })
        return
      }
      if (hasApiErrorReason(error, "ENGINE_CONFIG_INVALID")) {
        // Keep the draft intact: the user must explicitly review the refreshed Profile before retrying.
        setRequiresProfileReview(true)
        void loadScanWorkflowProfile(scanWorkflow).catch(() => undefined)
      }
    }
  }, [
    bulkInitiateScanMutation,
    configuration,
    initiateScanMutation,
    isYamlValid,
    onOpenChange,
    onSuccess,
    organizationId,
    organizationIds,
    resetDialogState,
    selectMode,
    selectedAgentID,
    inputSource,
    selectedWorkflowNames,
    selectedWorkflows,
    selectedWorkflowWithEngines,
    targetIds,
    tToast,
    targetId,
    loadScanWorkflowProfile,
    locale,
  ])

  const handleOpenChange = useCallback((newOpen: boolean) => {
    if (!initiateScanMutation.isPending && !bulkInitiateScanMutation.isPending) {
      onOpenChange(newOpen)
      if (!newOpen) {
        resetDialogState()
      }
    }
  }, [bulkInitiateScanMutation.isPending, initiateScanMutation.isPending, onOpenChange, resetDialogState])

  const hasConfig = configuration.trim().length > 0
  const hasNoEnabledSteps = hasNoEnabledWorkflowSteps(configuration)
  const canProceedToReview = selectedWorkflowNames.length > 0 && !isWorkflowConfigLoading && !engineCatalog.isLoading && !engineCatalog.isError
  const canStart = !requiresProfileReview && configuration.trim().length > 0 &&
    isYamlValid &&
    !hasNoEnabledSteps &&
    selectedWorkflowNames.length > 0 &&
    Boolean(selectedWorkflowWithEngines) &&
    !engineCatalog.isError

  return {
    workflows,
    isLoadingWorkflows,
    isWorkflowsError,
    selectedWorkflowNames,
    selectedScanWorkflowName: selectedWorkflowNames[0] ?? null,
    selectedPresetId: null,
    selectedAgentID,
    setSelectedAgentID,
    inputSource,
    setInputSource,
    selectMode,
    isSubmitting: initiateScanMutation.isPending || bulkInitiateScanMutation.isPending,
    currentStep,
    configuration,
    isConfigEdited,
    isYamlValid,
    showOverwriteConfirm,
    selectedWorkflows,
    engineCatalogDetails: engineCatalog.data,
    isEngineCatalogLoading: engineCatalog.isLoading,
    isEngineCatalogError: engineCatalog.isError,
    selectedScanWorkflows: selectedWorkflows,
    selectedPreset: null,
    hasConfig,
    hasNoEnabledSteps,
    canProceedToReview,
    canStart,
    configValidationRef,
    setCurrentStep,
    handleConfigSync,
    handleManualConfigChange,
    handleResetWorkflowConfig,
    handleWorkflowNamesChange,
    handleScanWorkflowNameChange: handleWorkflowNamesChange,
    loadScanWorkflowProfile,
    requiresProfileReview,
    handleOverwriteConfirm,
    handleOverwriteCancel,
    handleYamlValidationChange,
    handleInitiate,
    handleOpenChange,
    setShowOverwriteConfirm,
  }
}
