import * as React from "react"
import { toast } from "sonner"
import { useQueryClient } from "@tanstack/react-query"
import { quickScan } from "@/services/scan.service"
import { TargetValidator } from "@/lib/target-validator"
import { useEngines } from "@/hooks/use-engines"

type UseQuickScanDialogStateProps = {
  t: (key: string, params?: Record<string, string | number | Date>) => string
}

export function useQuickScanDialogState({ t }: UseQuickScanDialogStateProps) {
  const [open, setOpen] = React.useState(false)
  const [isSubmitting, setIsSubmitting] = React.useState(false)
  const [step, setStep] = React.useState(1)
  const queryClient = useQueryClient()

  const [targetInput, setTargetInput] = React.useState("")
  const [selectedEngineIds, setSelectedEngineIds] = React.useState<number[]>([])
  const [selectedPresetId, setSelectedPresetId] = React.useState<string | null>(null)

  const [configuration, setConfiguration] = React.useState("")
  const [isConfigEdited, setIsConfigEdited] = React.useState(false)
  const [isYamlValid, setIsYamlValid] = React.useState(true)
  const [showOverwriteConfirm, setShowOverwriteConfirm] = React.useState(false)
  const [pendingConfigChange, setPendingConfigChange] = React.useState<string | null>(null)

  const { data: engines } = useEngines()
  const lineNumbersRef = React.useRef<HTMLDivElement | null>(null)

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

  const selectedEngines = React.useMemo(() => {
    if (!selectedEngineIds.length || !engines) return []
    return engines.filter((engine) => selectedEngineIds.includes(engine.id))
  }, [selectedEngineIds, engines])

  const resetForm = React.useCallback(() => {
    setTargetInput("")
    setSelectedEngineIds([])
    setSelectedPresetId(null)
    setConfiguration("")
    setIsConfigEdited(false)
    setIsYamlValid(true)
    setPendingConfigChange(null)
    setShowOverwriteConfirm(false)
    setStep(1)
  }, [])

  const handleClose = React.useCallback((isOpen: boolean) => {
    setOpen(isOpen)
    if (!isOpen) resetForm()
  }, [resetForm])

  const handlePresetConfigChange = React.useCallback((value: string) => {
    if (isConfigEdited && configuration !== value) {
      setPendingConfigChange(value)
      setShowOverwriteConfirm(true)
    } else {
      setConfiguration(value)
      setIsConfigEdited(false)
    }
  }, [configuration, isConfigEdited])

  const handleManualConfigChange = React.useCallback((value: string) => {
    setConfiguration(value)
    setIsConfigEdited(true)
  }, [])

  const handleEngineIdsChange = React.useCallback((engineIds: number[]) => {
    setSelectedEngineIds(engineIds)
  }, [])

  const handleOverwriteConfirm = React.useCallback(() => {
    if (pendingConfigChange !== null) {
      setConfiguration(pendingConfigChange)
      setIsConfigEdited(false)
    }
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
  }, [pendingConfigChange])

  const handleOverwriteCancel = React.useCallback(() => {
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
  }, [])

  const handleYamlValidationChange = React.useCallback((isValid: boolean) => {
    setIsYamlValid(isValid)
  }, [])

  const canProceedToStep2 = validInputs.length > 0 && !hasErrors
  const canProceedToStep3 = selectedPresetId !== null && selectedEngineIds.length > 0
  const canSubmit = selectedEngineIds.length > 0 && configuration.trim().length > 0 && isYamlValid

  const handleNext = React.useCallback(() => {
    if (step === 1 && canProceedToStep2) setStep(2)
    else if (step === 2 && canProceedToStep3) setStep(3)
  }, [canProceedToStep2, canProceedToStep3, step])

  const handleBack = React.useCallback(() => {
    if (step > 1) setStep(step - 1)
  }, [step])

  const handleSubmit = React.useCallback(async () => {
    if (validInputs.length === 0) {
      toast.error(t("toast.noValidTarget"))
      return
    }
    if (hasErrors) {
      toast.error(t("toast.hasInvalidInputs", { count: invalidInputs.length }))
      return
    }
    if (selectedEngineIds.length === 0) {
      toast.error(t("toast.selectEngine"))
      return
    }
    if (!configuration.trim()) {
      toast.error(t("toast.emptyConfig"))
      return
    }

    const targets = validInputs.map((result) => result.originalInput)

    setIsSubmitting(true)
    try {
      const response = await quickScan({
        targets: targets.map((name) => ({ name })),
        configuration,
        engineIds: selectedEngineIds,
        engineNames: selectedEngines.map((engine) => engine.name),
      })

      const { targetStats, scans, count } = response
      const scanCount = scans?.length || count || 0

      toast.success(t("toast.createSuccess", { count: scanCount }), {
        description: targetStats.failed > 0
          ? t("toast.createSuccessDesc", { created: targetStats.created, failed: targetStats.failed })
          : undefined,
      })
      queryClient.invalidateQueries({ queryKey: ["scans"] })
      queryClient.invalidateQueries({ queryKey: ["scan-statistics"] })
      handleClose(false)
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: { code?: string; message?: string }; detail?: string } } }
      toast.error(err?.response?.data?.detail || err?.response?.data?.error?.message || t("toast.createFailed"))
    } finally {
      setIsSubmitting(false)
    }
  }, [
    configuration,
    handleClose,
    hasErrors,
    invalidInputs.length,
    queryClient,
    selectedEngineIds,
    selectedEngines,
    t,
    validInputs,
  ])

  return {
    open,
    handleClose,
    isSubmitting,
    step,
    targetInput,
    setTargetInput,
    selectedEngineIds,
    selectedPresetId,
    setSelectedPresetId,
    configuration,
    isConfigEdited,
    isYamlValid,
    showOverwriteConfirm,
    setShowOverwriteConfirm,
    lineNumbersRef,
    handleTextareaScroll,
    validInputs,
    invalidInputs,
    hasErrors,
    engines,
    selectedEngines,
    handlePresetConfigChange,
    handleManualConfigChange,
    handleEngineIdsChange,
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
