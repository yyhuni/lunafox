import { useCallback, useMemo, useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { mergeEngineConfigurations } from "@/lib/engine-config"
import {
  getApiErrorMessage,
  getInitiateScanValidationIssue,
  type InitiateScanSelectMode,
} from "@/lib/initiate-scan-helpers"
import { initiateScan } from "@/services/scan.service"
import { useEngines, usePresetEngines } from "@/hooks/use-engines"

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
  const [selectedEngineIds, setSelectedEngineIds] = useState<number[]>([])
  const [selectedPresetId, setSelectedPresetId] = useState<string | null>(null)
  const [selectMode, setSelectMode] = useState<InitiateScanSelectMode>("preset")
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [currentStep, setCurrentStep] = useState(1)

  const [configuration, setConfiguration] = useState("")
  const [isConfigEdited, setIsConfigEdited] = useState(false)
  const [isYamlValid, setIsYamlValid] = useState(true)
  const [showOverwriteConfirm, setShowOverwriteConfirm] = useState(false)
  const [pendingConfigChange, setPendingConfigChange] = useState<string | null>(null)
  const [pendingEngineIds, setPendingEngineIds] = useState<number[] | null>(null)
  const [pendingPresetId, setPendingPresetId] = useState<string | null>(null)

  const { data: engines, isLoading: isLoadingEngines, isError: isEnginesError } = useEngines()
  const { data: presetEngines, isLoading: isLoadingPresets, isError: isPresetsError } = usePresetEngines()

  const selectedEngines = useMemo(() => {
    if (!selectedEngineIds.length || !engines) return []
    return engines.filter((e) => selectedEngineIds.includes(e.id))
  }, [selectedEngineIds, engines])

  const selectedPreset = useMemo(() => {
    if (!presetEngines || !selectedPresetId) return null
    return presetEngines.find((preset) => preset.id === selectedPresetId) || null
  }, [presetEngines, selectedPresetId])

  const handleManualConfigChange = useCallback((value: string) => {
    setConfiguration(value)
    setIsConfigEdited(true)
  }, [])

  const buildConfigFromEngines = useCallback((engineIds: number[]) => {
    if (!engines) return ""
    const selected = engines.filter((e) => engineIds.includes(e.id))
    return mergeEngineConfigurations(selected.map((e) => e.configuration || ""))
  }, [engines])

  const applyEngineSelection = useCallback((engineIds: number[], nextConfig: string) => {
    setSelectedEngineIds(engineIds)
    setConfiguration(nextConfig)
    setIsConfigEdited(false)
    setIsYamlValid(true)
  }, [])

  const handleEngineIdsChange = useCallback((engineIds: number[]) => {
    const nextConfig = buildConfigFromEngines(engineIds)
    if (isConfigEdited && configuration !== nextConfig) {
      setPendingEngineIds(engineIds)
      setPendingConfigChange(nextConfig)
      setPendingPresetId(null)
      setShowOverwriteConfirm(true)
      return
    }
    applyEngineSelection(engineIds, nextConfig)
    setSelectedPresetId(null)
  }, [applyEngineSelection, buildConfigFromEngines, configuration, isConfigEdited])

  const handlePresetSelect = useCallback((presetId: string, presetConfig: string) => {
    if (isConfigEdited && configuration !== presetConfig) {
      setPendingEngineIds(null)
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
        const nextEngineIds = pendingEngineIds ?? selectedEngineIds
        applyEngineSelection(nextEngineIds, pendingConfigChange)
      }
    }
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
    setPendingEngineIds(null)
    setPendingPresetId(null)
  }, [applyEngineSelection, pendingConfigChange, pendingEngineIds, pendingPresetId, selectedEngineIds])

  const handleOverwriteCancel = useCallback(() => {
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
    setPendingEngineIds(null)
    setPendingPresetId(null)
  }, [])

  const handleYamlValidationChange = useCallback((isValid: boolean) => {
    setIsYamlValid(isValid)
  }, [])

  const handleSelectModeChange = useCallback((value: string) => {
    setSelectMode(value === "custom" ? "custom" : "preset")
  }, [])

  const resetDialogState = useCallback(() => {
    setSelectedEngineIds([])
    setSelectedPresetId(null)
    setConfiguration("")
    setIsConfigEdited(false)
    setIsYamlValid(true)
    setCurrentStep(1)
    setSelectMode("preset")
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
    setPendingEngineIds(null)
    setPendingPresetId(null)
  }, [])

  const handleInitiate = useCallback(async () => {
    const issue = getInitiateScanValidationIssue({
      selectMode,
      selectedPresetId,
      selectedEngineIds,
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
      const engineIds = selectMode === "custom" ? selectedEngineIds : []
      const engineNames = selectMode === "custom"
        ? selectedEngines.slice(0, 1).map((e) => e.name)
        : [selectedPresetId as string]

      const response = await initiateScan({
        organizationId,
        targetId,
        configuration,
        engineIds,
        engineNames,
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
    selectedEngineIds,
    selectedEngines,
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
    : selectedEngineIds.length > 0
  const canStart = configuration.trim().length > 0 &&
    isYamlValid &&
    (selectMode === "preset" ? !!selectedPresetId : selectedEngineIds.length > 0)

  return {
    engines,
    isLoadingEngines,
    isEnginesError,
    presetEngines,
    isLoadingPresets,
    isPresetsError,
    selectedEngineIds,
    selectedPresetId,
    selectMode,
    isSubmitting,
    currentStep,
    configuration,
    isConfigEdited,
    isYamlValid,
    showOverwriteConfirm,
    selectedEngines,
    selectedPreset,
    hasConfig,
    canProceedToReview,
    canStart,
    setCurrentStep,
    handleManualConfigChange,
    handleEngineIdsChange,
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
