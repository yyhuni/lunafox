import * as React from "react"
import { toastFeedback } from "@/lib/toast-helpers"
import { TargetValidator } from "@/lib/target-validator"
import {
  adaptWorkflowProfile,
  hasNoEnabledWorkflowSteps,
  serializeCanonicalWorkflowConfiguration,
  serializeWorkflowProfileDraft,
} from "@/lib/workflow-config"
import { hasApiErrorReason } from "@/lib/api-error-info"
import { useLoadScanWorkflowProfile, useScanWorkflows } from "@/hooks/use-scan-workflows"
import { useQuickScan } from "@/hooks/use-scans"
import { useEngineCatalogDetails } from "@/hooks/use-engine-catalog"
import { buildWorkflowWithEngineCatalog } from "@/lib/engine-catalog"
import type { Locale } from "@/i18n/config"
import type { ScanInputSource } from "@/types/scan.types"

type UseQuickScanDialogStateProps = {
  t: (key: string, params?: Record<string, string | number | Date>) => string
  locale?: Locale
}

export function useQuickScanDialogState({ t, locale = "en" }: UseQuickScanDialogStateProps) {
  const [open, setOpen] = React.useState(false)
  const [step, setStep] = React.useState(1)
  const quickScanMutation = useQuickScan()
  const loadScanWorkflowProfile = useLoadScanWorkflowProfile()

  const [targetInput, setTargetInput] = React.useState("")
  const [selectedWorkflowNames, setSelectedWorkflowNames] = React.useState<string[]>([])
  const [selectedAgentID, setSelectedAgentID] = React.useState<number | null>(null)
  const [inputSource, setInputSource] = React.useState<ScanInputSource>("scanSnapshot")
  const selectMode = "custom" as const

  const [configuration, setConfiguration] = React.useState("")
  const [isConfigEdited, setIsConfigEdited] = React.useState(false)
  const [isYamlValid, setIsYamlValid] = React.useState(true)
  const [showOverwriteConfirm, setShowOverwriteConfirm] = React.useState(false)
  const [pendingConfigChange, setPendingConfigChange] = React.useState<string | null>(null)
  const [pendingWorkflowNames, setPendingWorkflowNames] = React.useState<string[] | null>(null)
  const [isWorkflowConfigLoading, setIsWorkflowConfigLoading] = React.useState(false)
  const [requiresProfileReview, setRequiresProfileReview] = React.useState(false)
  const workflowSelectionRequestRef = React.useRef(0)

  const { data: workflows, isLoading: isLoadingWorkflows, isError: isWorkflowsError } = useScanWorkflows()
  const lineNumbersRef = React.useRef<HTMLDivElement | null>(null)
  const textareaRef = React.useRef<HTMLTextAreaElement | null>(null)

  const handleTextareaScroll = React.useCallback((event: React.UIEvent<HTMLTextAreaElement>) => {
    if (lineNumbersRef.current) {
      lineNumbersRef.current.scrollTop = event.currentTarget.scrollTop
    }
  }, [])

  const validationResults = React.useMemo(() => {
    const lines = targetInput.split("\n")
    return TargetValidator.validateInputBatch(lines)
  }, [targetInput])

  const validInputs = React.useMemo(
    () => validationResults.filter((result) => result.isValid && !result.isEmptyLine),
    [validationResults]
  )
  const invalidInputs = React.useMemo(
    () => validationResults.filter((result) => !result.isValid),
    [validationResults]
  )
  const hasErrors = invalidInputs.length > 0

  const selectedWorkflows = React.useMemo(() => {
    if (!selectedWorkflowNames.length || !workflows) return []
    const selectedSet = new Set(selectedWorkflowNames)
    return workflows.filter((item) => selectedSet.has(item.name))
  }, [selectedWorkflowNames, workflows])
  const selectedEngineIds = React.useMemo(
    () => selectedWorkflows.flatMap((workflow) => workflow.steps?.map((step) => step.engineId) ?? []),
    [selectedWorkflows]
  )
  const engineCatalog = useEngineCatalogDetails(selectedEngineIds)

  const selectedWorkflowWithEngines = React.useMemo(() => {
    const workflow = selectedWorkflows[0]
    if (!workflow || !engineCatalog.data) return undefined
    try {
      return buildWorkflowWithEngineCatalog(workflow, engineCatalog.data, locale)
    } catch {
      return undefined
    }
  }, [engineCatalog.data, locale, selectedWorkflows])

  const resetForm = React.useCallback(() => {
    workflowSelectionRequestRef.current += 1
    setTargetInput("")
    setSelectedWorkflowNames([])
    setSelectedAgentID(null)
    setInputSource("scanSnapshot")
    setConfiguration("")
    setIsConfigEdited(false)
    setIsYamlValid(true)
    setPendingConfigChange(null)
    setPendingWorkflowNames(null)
    setShowOverwriteConfirm(false)
    setIsWorkflowConfigLoading(false)
    setRequiresProfileReview(false)
    setStep(1)
  }, [])

  const handleClose = React.useCallback((isOpen: boolean) => {
    setOpen(isOpen)
    if (!isOpen) resetForm()
  }, [resetForm])

  const handleConfigSync = React.useCallback((value: string) => {
    setConfiguration(value)
  }, [])

  const handleManualConfigChange = React.useCallback((value: string) => {
    setConfiguration(value)
    setIsConfigEdited(true)
  }, [])

  const loadProfileConfiguration = React.useCallback(async (workflowName: string) => {
    const profile = await loadScanWorkflowProfile(workflowName)
    const workflow = workflows?.find((item) => item.name === workflowName)
    if (!workflow) throw new Error("Selected Workflow is unavailable")
    const draft = adaptWorkflowProfile(profile, workflow)
    return serializeWorkflowProfileDraft(draft)
  }, [loadScanWorkflowProfile, workflows])

  const applyWorkflowSelection = React.useCallback((workflowNames: string[], nextConfig: string) => {
    setSelectedWorkflowNames(workflowNames)
    setConfiguration(nextConfig)
    setIsConfigEdited(false)
    setIsYamlValid(true)
    setIsWorkflowConfigLoading(false)
    setRequiresProfileReview(false)
  }, [])

  const handleResetWorkflowConfig = React.useCallback(async () => {
    const workflowName = selectedWorkflowNames[0]
    if (!workflowName) return
    const nextConfig = await loadProfileConfiguration(workflowName)
    setConfiguration(nextConfig)
    setIsConfigEdited(false)
    setIsYamlValid(true)
    setRequiresProfileReview(false)
  }, [loadProfileConfiguration, selectedWorkflowNames])

  const handleScanWorkflowNameChange = React.useCallback(async (workflowNames: string[]) => {
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

  React.useEffect(() => {
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

    void handleScanWorkflowNameChange([firstWorkflow.name])
  }, [handleScanWorkflowNameChange, isLoadingWorkflows, isWorkflowsError, open, selectedWorkflowNames.length, workflows])

  const handleOverwriteConfirm = React.useCallback(() => {
    if (pendingConfigChange !== null) {
      const nextWorkflowNames = pendingWorkflowNames ?? selectedWorkflowNames
      applyWorkflowSelection(nextWorkflowNames, pendingConfigChange)
    }
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
    setPendingWorkflowNames(null)
  }, [applyWorkflowSelection, pendingConfigChange, pendingWorkflowNames, selectedWorkflowNames])

  const handleOverwriteCancel = React.useCallback(() => {
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
    setPendingWorkflowNames(null)
  }, [])

  const handleYamlValidationChange = React.useCallback((isValid: boolean) => {
    setIsYamlValid(isValid)
  }, [])

  const canProceedToStep2 = validInputs.length > 0 && !hasErrors
  const canProceedToStep3 = selectedWorkflowNames.length > 0 && !isWorkflowConfigLoading && Boolean(selectedWorkflowWithEngines) && !engineCatalog.isLoading && !engineCatalog.isError
  const hasConfig = configuration.trim().length > 0
  const hasNoEnabledSteps = hasNoEnabledWorkflowSteps(configuration)
  const canSubmit = !requiresProfileReview && selectedWorkflowNames.length > 0 && configuration.trim().length > 0 && isYamlValid && !hasNoEnabledSteps && Boolean(selectedWorkflowWithEngines) && !engineCatalog.isError

  const handleNext = React.useCallback(() => {
    if (step === 1 && canProceedToStep2) setStep(2)
    else if (step === 2 && canProceedToStep3) setStep(3)
  }, [canProceedToStep2, canProceedToStep3, step])

  const handleBack = React.useCallback(() => {
    if (step > 1) setStep(step - 1)
  }, [step])

  const handleSubmit = React.useCallback(async () => {
    if (validInputs.length === 0) {
      toastFeedback.error(t("toast.noValidTarget"))
      return
    }
    if (hasErrors) {
      toastFeedback.error(t("toast.hasInvalidInputs", { count: invalidInputs.length }))
      return
    }
    if (selectedWorkflowNames.length === 0) {
      toastFeedback.error(t("toast.selectWorkflow"))
      return
    }
    if (!configuration.trim()) {
      toastFeedback.error(t("toast.emptyConfig"))
      return
    }
    if (hasNoEnabledWorkflowSteps(configuration)) {
      toastFeedback.error(t("toast.noEnabledSteps"))
      return
    }

    const targets = validInputs.map((result) => result.originalInput)
    const scanWorkflow = selectedWorkflowNames[0]

    if (!scanWorkflow) {
      toastFeedback.error(t("toast.selectWorkflow"))
      return
    }

    try {
      const selectedWorkflow = selectedWorkflows[0]
      if (!selectedWorkflow) {
        toastFeedback.error(t("toast.selectWorkflow"))
        return
      }
      if (!selectedWorkflowWithEngines) {
        throw new Error("Selected Workflow Engine catalog is unavailable")
      }
      await quickScanMutation.mutateAsync({
        targets: targets.map((name) => ({ name })),
        configuration: serializeCanonicalWorkflowConfiguration(
          configuration,
          selectedWorkflow,
          selectedWorkflowWithEngines,
        ),
        scanWorkflow,
        agentId: selectedAgentID ?? undefined,
        inputSource,
      })
      handleClose(false)
    } catch (error) {
      if (error instanceof Error && error.name === "WorkflowConfigurationDraftError") {
        toastFeedback.error(t("toast.invalidConfig"), { description: error.message })
        return
      }
      if (hasApiErrorReason(error, "ENGINE_CONFIG_INVALID")) {
        setRequiresProfileReview(true)
        void loadScanWorkflowProfile(scanWorkflow).catch(() => undefined)
      }
    }
  }, [
    configuration,
    handleClose,
    hasErrors,
    invalidInputs.length,
    quickScanMutation,
    selectedWorkflowNames,
    selectedWorkflows,
    selectedWorkflowWithEngines,
    selectedAgentID,
    inputSource,
    t,
    validInputs,
    loadScanWorkflowProfile,
    locale,
  ])

  return {
    open,
    handleClose,
    isSubmitting: quickScanMutation.isPending,
    step,
    targetInput,
    setTargetInput,
    selectedWorkflowNames,
    selectedAgentID,
    setSelectedAgentID,
    inputSource,
    setInputSource,
    selectMode,
    configuration,
    isConfigEdited,
    isYamlValid,
    showOverwriteConfirm,
    setShowOverwriteConfirm,
    lineNumbersRef,
    textareaRef,
    handleTextareaScroll,
    validInputs,
    invalidInputs,
    hasErrors,
    workflows,
    isLoadingWorkflows,
    isWorkflowsError,
    selectedWorkflows,
    engineCatalogDetails: engineCatalog.data,
    isEngineCatalogLoading: engineCatalog.isLoading,
    isEngineCatalogError: engineCatalog.isError,
    hasConfig,
    hasNoEnabledSteps,
    requiresProfileReview,
    handleConfigSync,
    handleManualConfigChange,
    handleResetWorkflowConfig,
    handleScanWorkflowNameChange,
    handleOverwriteConfirm,
    handleOverwriteCancel,
    handleYamlValidationChange,
    canProceedToStep2,
    canProceedToStep3,
    canSubmit,
    handleNext,
    handleBack,
    handleSubmit,
    totalSteps: 3,
  }
}
