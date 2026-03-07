import { useCallback, useMemo, useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import {
  extractWorkflowIds,
  mergeWorkflowConfigurations,
  parseWorkflowConfiguration,
  serializeWorkflowConfiguration,
} from "@/lib/workflow-config"
import {
  getApiErrorMessage,
  getInitiateScanValidationIssue,
  type InitiateScanSelectMode,
} from "@/lib/initiate-scan-helpers"
import { initiateScan } from "@/services/scan.service"
import { useWorkflows, useWorkflowProfiles } from "@/hooks/use-workflows"
import type { WorkflowProfile } from "@/types/workflow.types"

type UseInitiateScanDialogStateProps = {
  organizationId?: number
  targetId?: number
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
  tToast: (key: string, params?: Record<string, string | number | Date>) => string
}

function resolveWorkflowProfileNames(preset: WorkflowProfile | null): string[] {
  if (!preset) return []
  const preferred = preset.workflowIds || preset.workflowNames || []
  if (preferred.length > 0) {
    return preferred.filter((item) => item && item.trim().length > 0)
  }
  return extractWorkflowIds(preset.configuration)
}

export function useInitiateScanDialogState({
  organizationId,
  targetId,
  onOpenChange,
  onSuccess,
  tToast,
}: UseInitiateScanDialogStateProps) {
  const queryClient = useQueryClient()
  const [selectedWorkflowNames, setSelectedWorkflowNames] = useState<string[]>([])
  const [selectedPresetId, setSelectedPresetId] = useState<string | null>(null)
  const [selectMode, setSelectMode] = useState<InitiateScanSelectMode>("preset")
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [currentStep, setCurrentStep] = useState(1)

  const [configuration, setConfiguration] = useState("")
  const [isConfigEdited, setIsConfigEdited] = useState(false)
  const [isYamlValid, setIsYamlValid] = useState(true)
  const [showOverwriteConfirm, setShowOverwriteConfirm] = useState(false)
  const [pendingConfigChange, setPendingConfigChange] = useState<string | null>(null)
  const [pendingWorkflowNames, setPendingWorkflowNames] = useState<string[] | null>(null)
  const [pendingPresetId, setPendingPresetId] = useState<string | null>(null)

  const { data: workflows, isLoading: isLoadingWorkflows, isError: isWorkflowsError } = useWorkflows()
  const { data: presetWorkflows, isLoading: isLoadingPresets, isError: isPresetsError } = useWorkflowProfiles()

  const selectedWorkflows = useMemo(() => {
    if (!selectedWorkflowNames.length || !workflows) return []
    const selectedSet = new Set(selectedWorkflowNames)
    return workflows.filter((item) => selectedSet.has(item.name))
  }, [selectedWorkflowNames, workflows])

  const selectedPreset = useMemo(() => {
    if (!presetWorkflows || !selectedPresetId) return null
    return presetWorkflows.find((preset) => preset.id === selectedPresetId) || null
  }, [presetWorkflows, selectedPresetId])

  const handleManualConfigChange = useCallback((value: string) => {
    setConfiguration(value)
    setIsConfigEdited(true)
  }, [])

  const buildConfigFromWorkflows = useCallback((workflowNames: string[]) => {
    if (!workflows) return ""
    const selectedSet = new Set(workflowNames)
    const selected = workflows.filter((item) => selectedSet.has(item.name))
    return serializeWorkflowConfiguration(mergeWorkflowConfigurations(selected.map((item) => item.configuration)))
  }, [workflows])

  const applyWorkflowSelection = useCallback((workflowNames: string[], nextConfig: string) => {
    setSelectedWorkflowNames(workflowNames)
    setConfiguration(nextConfig)
    setIsConfigEdited(false)
    setIsYamlValid(true)
  }, [])

  const handleWorkflowNamesChange = useCallback((workflowNames: string[]) => {
    const nextConfig = buildConfigFromWorkflows(workflowNames)
    if (isConfigEdited && configuration !== nextConfig) {
      setPendingWorkflowNames(workflowNames)
      setPendingConfigChange(nextConfig)
      setPendingPresetId(null)
      setShowOverwriteConfirm(true)
      return
    }
    applyWorkflowSelection(workflowNames, nextConfig)
    setSelectedPresetId(null)
  }, [applyWorkflowSelection, buildConfigFromWorkflows, configuration, isConfigEdited])

  const handlePresetSelect = useCallback((presetId: string, presetConfig: string) => {
    if (isConfigEdited && configuration !== presetConfig) {
      setPendingWorkflowNames(null)
      setPendingConfigChange(presetConfig)
      setPendingPresetId(presetId)
      setShowOverwriteConfirm(true)
      return
    }
    setSelectedPresetId(presetId)
    setConfiguration(presetConfig)
    setIsConfigEdited(false)
    setIsYamlValid(true)
  }, [configuration, isConfigEdited])

  const handleOverwriteConfirm = useCallback(() => {
    if (pendingConfigChange !== null) {
      if (pendingPresetId !== null) {
        setSelectedPresetId(pendingPresetId)
        setConfiguration(pendingConfigChange)
        setIsConfigEdited(false)
        setIsYamlValid(true)
      } else {
        const nextWorkflowNames = pendingWorkflowNames ?? selectedWorkflowNames
        applyWorkflowSelection(nextWorkflowNames, pendingConfigChange)
      }
    }
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
    setPendingWorkflowNames(null)
    setPendingPresetId(null)
  }, [applyWorkflowSelection, pendingConfigChange, pendingPresetId, pendingWorkflowNames, selectedWorkflowNames])

  const handleOverwriteCancel = useCallback(() => {
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
    setPendingWorkflowNames(null)
    setPendingPresetId(null)
  }, [])

  const handleYamlValidationChange = useCallback((isValid: boolean) => {
    setIsYamlValid(isValid)
  }, [])

  const handleSelectModeChange = useCallback((value: string) => {
    setSelectMode(value === "custom" ? "custom" : "preset")
  }, [])

  const resetDialogState = useCallback(() => {
    setSelectedWorkflowNames([])
    setSelectedPresetId(null)
    setConfiguration("")
    setIsConfigEdited(false)
    setIsYamlValid(true)
    setCurrentStep(1)
    setSelectMode("preset")
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
    setPendingWorkflowNames(null)
    setPendingPresetId(null)
  }, [])

  const handleInitiate = useCallback(async () => {
    const issue = getInitiateScanValidationIssue({
      selectMode,
      selectedPresetId,
      selectedWorkflowNames,
      configuration,
      isYamlValid,
      organizationId,
      targetId,
    })
    if (issue) {
      toast.error(
        tToast(issue.titleKey),
        issue.descriptionKey ? { description: tToast(issue.descriptionKey) } : undefined
      )
      return
    }

    const workflowNames = selectMode === "custom"
      ? selectedWorkflowNames
      : resolveWorkflowProfileNames(selectedPreset)
    if (workflowNames.length === 0) {
      toast.error(tToast("noWorkflowSelected"))
      return
    }

    setIsSubmitting(true)
    try {
      const response = await initiateScan({
        organizationId,
        targetId,
        configuration: parseWorkflowConfiguration(configuration),
        workflowNames,
      })

      const scanCount = response.scans?.length || response.count || 0
      toast.success(tToast("scanInitiated"), {
        description: response.message || tToast("scanInitiatedDesc", { count: scanCount }),
      })
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["scans"] }),
        queryClient.invalidateQueries({ queryKey: ["scan-statistics"] }),
      ])
      onSuccess?.()
      onOpenChange(false)
      resetDialogState()
    } catch (err: unknown) {
      const message = getApiErrorMessage(err) ?? tToast("unknownError")
      toast.error(tToast("initiateScanFailed"), {
        description: message,
      })
    } finally {
      setIsSubmitting(false)
    }
  }, [
    configuration,
    isYamlValid,
    onOpenChange,
    onSuccess,
    organizationId,
    queryClient,
    resetDialogState,
    selectMode,
    selectedPreset,
    selectedPresetId,
    selectedWorkflowNames,
    tToast,
    targetId,
  ])

  const handleOpenChange = useCallback((newOpen: boolean) => {
    if (!isSubmitting) {
      onOpenChange(newOpen)
      if (!newOpen) {
        resetDialogState()
      }
    }
  }, [isSubmitting, onOpenChange, resetDialogState])

  const hasConfig = configuration.trim().length > 0
  const canProceedToReview = selectMode === "preset"
    ? !!selectedPresetId
    : selectedWorkflowNames.length > 0
  const canStart = configuration.trim().length > 0 &&
    isYamlValid &&
    (selectMode === "preset" ? !!selectedPresetId : selectedWorkflowNames.length > 0)

  return {
    workflows,
    isLoadingWorkflows,
    isWorkflowsError,
    presetWorkflows,
    isLoadingPresets,
    isPresetsError,
    selectedWorkflowNames,
    selectedPresetId,
    selectMode,
    isSubmitting,
    currentStep,
    configuration,
    isConfigEdited,
    isYamlValid,
    showOverwriteConfirm,
    selectedWorkflows,
    selectedPreset,
    hasConfig,
    canProceedToReview,
    canStart,
    setCurrentStep,
    handleManualConfigChange,
    handleWorkflowNamesChange,
    handlePresetSelect,
    handleOverwriteConfirm,
    handleOverwriteCancel,
    handleYamlValidationChange,
    handleSelectModeChange,
    handleInitiate,
    handleOpenChange,
    setShowOverwriteConfirm,
  }
}
