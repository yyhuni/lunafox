import { useCallback, useMemo, useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { mergeWorkflowConfigurations } from "@/lib/workflow-config"
import {
  getApiErrorMessage,
  getInitiateScanValidationIssue,
  type InitiateScanSelectMode,
} from "@/lib/initiate-scan-helpers"
import { initiateScan } from "@/services/scan.service"
import { useWorkflows, usePresetWorkflows } from "@/hooks/use-workflows"

type UseInitiateScanDialogStateProps = {
  organizationId?: number
  targetId?: number
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
  tToast: (key: string, params?: Record<string, string | number | Date>) => string
}

export function useInitiateScanDialogState({
  organizationId,
  targetId,
  onOpenChange,
  onSuccess,
  tToast,
}: UseInitiateScanDialogStateProps) {
  const queryClient = useQueryClient()
  const [selectedWorkflowIds, setSelectedWorkflowIds] = useState<number[]>([])
  const [selectedPresetId, setSelectedPresetId] = useState<string | null>(null)
  const [selectMode, setSelectMode] = useState<InitiateScanSelectMode>("preset")
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [currentStep, setCurrentStep] = useState(1)

  const [configuration, setConfiguration] = useState("")
  const [isConfigEdited, setIsConfigEdited] = useState(false)
  const [isYamlValid, setIsYamlValid] = useState(true)
  const [showOverwriteConfirm, setShowOverwriteConfirm] = useState(false)
  const [pendingConfigChange, setPendingConfigChange] = useState<string | null>(null)
  const [pendingWorkflowIds, setPendingWorkflowIds] = useState<number[] | null>(null)
  const [pendingPresetId, setPendingPresetId] = useState<string | null>(null)

  const { data: workflows, isLoading: isLoadingWorkflows, isError: isWorkflowsError } = useWorkflows()
  const { data: presetWorkflows, isLoading: isLoadingPresets, isError: isPresetsError } = usePresetWorkflows()

  const selectedWorkflows = useMemo(() => {
    if (!selectedWorkflowIds.length || !workflows) return []
    return workflows.filter((item) => selectedWorkflowIds.includes(item.id))
  }, [selectedWorkflowIds, workflows])

  const selectedPreset = useMemo(() => {
    if (!presetWorkflows || !selectedPresetId) return null
    return presetWorkflows.find((preset) => preset.id === selectedPresetId) || null
  }, [presetWorkflows, selectedPresetId])

  const handleManualConfigChange = useCallback((value: string) => {
    setConfiguration(value)
    setIsConfigEdited(true)
  }, [])

  const buildConfigFromWorkflows = useCallback((workflowIds: number[]) => {
    if (!workflows) return ""
    const selected = workflows.filter((item) => workflowIds.includes(item.id))
    return mergeWorkflowConfigurations(selected.map((item) => item.configuration || ""))
  }, [workflows])

  const applyWorkflowSelection = useCallback((workflowIds: number[], nextConfig: string) => {
    setSelectedWorkflowIds(workflowIds)
    setConfiguration(nextConfig)
    setIsConfigEdited(false)
    setIsYamlValid(true)
  }, [])

  const handleWorkflowIdsChange = useCallback((workflowIds: number[]) => {
    const nextConfig = buildConfigFromWorkflows(workflowIds)
    if (isConfigEdited && configuration !== nextConfig) {
      setPendingWorkflowIds(workflowIds)
      setPendingConfigChange(nextConfig)
      setPendingPresetId(null)
      setShowOverwriteConfirm(true)
      return
    }
    applyWorkflowSelection(workflowIds, nextConfig)
    setSelectedPresetId(null)
  }, [applyWorkflowSelection, buildConfigFromWorkflows, configuration, isConfigEdited])

  const handlePresetSelect = useCallback((presetId: string, presetConfig: string) => {
    if (isConfigEdited && configuration !== presetConfig) {
      setPendingWorkflowIds(null)
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
        const nextWorkflowIds = pendingWorkflowIds ?? selectedWorkflowIds
        applyWorkflowSelection(nextWorkflowIds, pendingConfigChange)
      }
    }
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
    setPendingWorkflowIds(null)
    setPendingPresetId(null)
  }, [applyWorkflowSelection, pendingConfigChange, pendingWorkflowIds, pendingPresetId, selectedWorkflowIds])

  const handleOverwriteCancel = useCallback(() => {
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
    setPendingWorkflowIds(null)
    setPendingPresetId(null)
  }, [])

  const handleYamlValidationChange = useCallback((isValid: boolean) => {
    setIsYamlValid(isValid)
  }, [])

  const handleSelectModeChange = useCallback((value: string) => {
    setSelectMode(value === "custom" ? "custom" : "preset")
  }, [])

  const resetDialogState = useCallback(() => {
    setSelectedWorkflowIds([])
    setSelectedPresetId(null)
    setConfiguration("")
    setIsConfigEdited(false)
    setIsYamlValid(true)
    setCurrentStep(1)
    setSelectMode("preset")
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
    setPendingWorkflowIds(null)
    setPendingPresetId(null)
  }, [])

  const handleInitiate = useCallback(async () => {
    const issue = getInitiateScanValidationIssue({
      selectMode,
      selectedPresetId,
      selectedWorkflowIds,
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

    setIsSubmitting(true)
    try {
      const workflowIds = selectMode === "custom" ? selectedWorkflowIds : []
      const workflowNames = selectMode === "custom"
        ? selectedWorkflows.slice(0, 1).map((item) => item.name)
        : [selectedPresetId as string]

      const response = await initiateScan({
        organizationId,
        targetId,
        configuration,
        workflowIds,
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
    selectedWorkflowIds,
    selectedWorkflows,
    selectedPresetId,
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
    : selectedWorkflowIds.length > 0
  const canStart = configuration.trim().length > 0 &&
    isYamlValid &&
    (selectMode === "preset" ? !!selectedPresetId : selectedWorkflowIds.length > 0)

  return {
    workflows,
    isLoadingWorkflows,
    isWorkflowsError,
    presetWorkflows,
    isLoadingPresets,
    isPresetsError,
    selectedWorkflowIds,
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
    handleWorkflowIdsChange,
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
