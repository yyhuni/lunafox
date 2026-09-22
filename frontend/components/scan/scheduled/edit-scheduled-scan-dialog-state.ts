import React from "react"
import cronstrue from "cronstrue/i18n"
import { toastFeedback } from "@/lib/toast-helpers"
import {
  getConfigConflictMessage,
  getNextCronExecutions,
  isCronExpressionValid,
  isIanaTimeZoneValid,
} from "@/lib/scheduled-scan-helpers"
import { useUpdateScheduledScan } from "@/hooks/use-scheduled-scans"
import { useTargets } from "@/hooks/use-targets"
import { useLoadScanWorkflowProfile, useScanWorkflows } from "@/hooks/use-scan-workflows"
import { useEngineCatalogDetails } from "@/hooks/use-engine-catalog"
import { scanWorkflowName } from "@/lib/resource-name"
import { buildWorkflowWithEngineCatalog } from "@/lib/engine-catalog"
import {
  adaptWorkflowProfile,
  serializeWorkflowProfileDraft,
  serializeWorkflowConfiguration,
  type WorkflowProfileDraft,
} from "@/lib/workflow-config"
import type { Locale } from "@/i18n/config"
import type { ScanConfigValidationHandle } from "@/components/scan/scan-config-view-toggle"
import type { EngineConfigFormValues } from "@/types/engine-config.types"
import type { ScheduledScan, UpdateScheduledScanRequest } from "@/types/scheduled-scan.types"
import type { ScanWorkflow } from "@/types/scan-workflow.types"
import type { Target } from "@/types/target.types"
import type { ScanInputSource } from "@/types/scan.types"

interface UseEditScheduledScanDialogStateProps {
  open: boolean
  scheduledScan: ScheduledScan | null
  locale?: Locale
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
  t: (key: string, params?: Record<string, string | number | Date>) => string
}

export function useEditScheduledScanDialogState({
  open,
  scheduledScan,
  locale = "en",
  onOpenChange,
  onSuccess,
  t,
}: UseEditScheduledScanDialogStateProps) {
  const { mutate: updateScheduledScan, isPending } = useUpdateScheduledScan()
  const { data: targetsData } = useTargets()
  const loadScanWorkflowProfile = useLoadScanWorkflowProfile()
  const {
    data: workflowsData,
    isLoading: isLoadingWorkflows,
    isError: isWorkflowsError,
  } = useScanWorkflows()
  const targets: Target[] = targetsData?.targets || []
  const workflows = React.useMemo<ScanWorkflow[]>(() => workflowsData || [], [workflowsData])

  const [displayName, setDisplayName] = React.useState("")
  const [scanWorkflow, setScanWorkflow] = React.useState("")
  const [selectedTargetId, setSelectedTargetId] = React.useState<number | null>(null)
  const [selectedAgentID, setSelectedAgentID] = React.useState<number | null>(null)
  const [inputSource, setInputSource] = React.useState<ScanInputSource>("scanSnapshot")
  const [timeZone, setTimeZone] = React.useState("")
  const [cronExpression, setCronExpression] = React.useState("")
  const [configuration, setConfiguration] = React.useState("")
  const [isConfigEdited, setIsConfigEdited] = React.useState(false)
  const [isYamlValid, setIsYamlValid] = React.useState(true)
  const [replacementConfiguration, setReplacementConfiguration] = React.useState<string | null>(null)
  const [isWorkflowConfigLoading, setIsWorkflowConfigLoading] = React.useState(false)
  const configValidationRef = React.useRef<ScanConfigValidationHandle | null>(null)
  const formValuesCacheRef = React.useRef<EngineConfigFormValues>({})
  const [workflowProfileDraft, setWorkflowProfileDraft] = React.useState<WorkflowProfileDraft | null>(null)
  const [isWorkflowProfileLoading, setIsWorkflowProfileLoading] = React.useState(false)
  const [profileBootstrapWorkflowName, setProfileBootstrapWorkflowName] = React.useState<string | null>(null)
  const workflowSelectionRequestRef = React.useRef(0)
  const profileBootstrapRequestRef = React.useRef(0)

  const selectedWorkflow = React.useMemo(
    () => workflows.find((item) => item.name === scanWorkflow),
    [scanWorkflow, workflows]
  )
  const selectedEngineIds = React.useMemo(
    () => open ? (selectedWorkflow?.steps?.map((step) => step.engineId) ?? []) : [],
    [open, selectedWorkflow]
  )
  const engineCatalog = useEngineCatalogDetails(selectedEngineIds)
  const selectedWorkflowWithEngines = React.useMemo(() => {
    if (!selectedWorkflow || !engineCatalog.data) return undefined
    try {
      return buildWorkflowWithEngineCatalog(selectedWorkflow, engineCatalog.data, locale)
    } catch {
      return undefined
    }
  }, [engineCatalog.data, locale, selectedWorkflow])

  const cronPresets = React.useMemo(() => [
    { label: t("presets.everyMinute"), value: "* * * * *" },
    { label: t("presets.every5Minutes"), value: "*/5 * * * *" },
    { label: t("presets.everyHour"), value: "0 * * * *" },
    { label: t("presets.daily2am"), value: "0 2 * * *" },
    { label: t("presets.daily4am"), value: "0 4 * * *" },
    { label: t("presets.weekly"), value: "0 2 * * 1" },
    { label: t("presets.monthly"), value: "0 2 1 * *" },
  ], [t])

  React.useEffect(() => {
    if (scheduledScan && open) {
      workflowSelectionRequestRef.current += 1
      profileBootstrapRequestRef.current += 1
      const persistedWorkflowName = scanWorkflowName(scheduledScan.scanWorkflow)
      setDisplayName(scheduledScan.displayName)
      setScanWorkflow(persistedWorkflowName)
      setSelectedTargetId(scheduledScan.targetId)
      setSelectedAgentID(scheduledScan.agentId ?? null)
      setInputSource(scheduledScan.inputSource)
      setTimeZone(scheduledScan.timeZone)
      setCronExpression(scheduledScan.cronExpression || "0 2 * * *")
      setConfiguration(serializeWorkflowConfiguration(scheduledScan.configuration))
      setIsConfigEdited(false)
      setIsYamlValid(true)
      setReplacementConfiguration(null)
      setIsWorkflowConfigLoading(false)
      formValuesCacheRef.current = {}
      setWorkflowProfileDraft(null)
      setProfileBootstrapWorkflowName(persistedWorkflowName || null)
      setIsWorkflowProfileLoading(Boolean(persistedWorkflowName))
    }
  }, [scheduledScan, open])

  React.useEffect(() => {
    if (!open || !scheduledScan || !profileBootstrapWorkflowName) return
    const workflow = workflows.find((item) => item.name === profileBootstrapWorkflowName)
    if (!workflow) {
      if (!isLoadingWorkflows) {
        setIsWorkflowProfileLoading(false)
        setProfileBootstrapWorkflowName(null)
      }
      return
    }

    const requestId = profileBootstrapRequestRef.current + 1
    profileBootstrapRequestRef.current = requestId
    setIsWorkflowProfileLoading(true)
    void Promise.resolve(loadScanWorkflowProfile(profileBootstrapWorkflowName))
      .then((profile) => {
        if (profileBootstrapRequestRef.current !== requestId) return
        setWorkflowProfileDraft(adaptWorkflowProfile(profile, workflow))
      })
      .catch(() => {
        // Keep persisted Schedule configuration intact. A missing Profile only
        // prevents disabled-Step UI draft recovery in the editor.
      })
      .finally(() => {
        if (profileBootstrapRequestRef.current !== requestId) return
        setIsWorkflowProfileLoading(false)
        setProfileBootstrapWorkflowName(null)
      })
  }, [isLoadingWorkflows, loadScanWorkflowProfile, open, profileBootstrapWorkflowName, scheduledScan, workflows])

  const handleWorkflowNamesChange = React.useCallback(async (workflowNames: string[]) => {
    const nextWorkflowName = workflowNames.at(-1)
    if (!nextWorkflowName || nextWorkflowName === scanWorkflow) return

    const requestId = workflowSelectionRequestRef.current + 1
    workflowSelectionRequestRef.current = requestId
    profileBootstrapRequestRef.current += 1
    setProfileBootstrapWorkflowName(null)
    setIsWorkflowProfileLoading(false)
    setIsWorkflowConfigLoading(true)
    try {
      const workflow = workflows.find((item) => item.name === nextWorkflowName)
      if (!workflow) throw new Error("Selected Workflow is unavailable")

      const profile = await loadScanWorkflowProfile(nextWorkflowName)
      const draft = adaptWorkflowProfile(profile, workflow)
      const nextConfiguration = serializeWorkflowProfileDraft(draft)
      if (workflowSelectionRequestRef.current !== requestId) return

      setScanWorkflow(nextWorkflowName)
      setConfiguration(nextConfiguration)
      setIsConfigEdited(false)
      setIsYamlValid(true)
      setReplacementConfiguration(nextConfiguration)
      formValuesCacheRef.current = {}
      setWorkflowProfileDraft(draft)
    } catch {
      // A Profile is the only valid source of a replacement configuration.
      // Keep the current Schedule state instead of submitting mismatched steps.
    } finally {
      if (workflowSelectionRequestRef.current === requestId) {
        setIsWorkflowConfigLoading(false)
      }
    }
  }, [loadScanWorkflowProfile, scanWorkflow, workflows])

  const handleConfigSync = React.useCallback((value: string) => {
    setConfiguration(value)
  }, [])

  const handleManualConfigChange = React.useCallback((value: string) => {
    setConfiguration(value)
    setIsConfigEdited(true)
  }, [])

  const handleYamlValidationChange = React.useCallback((isValid: boolean) => {
    setIsYamlValid(isValid)
  }, [])

  const handleResetWorkflowConfig = React.useCallback(async () => {
    if (!scanWorkflow) return

    setIsWorkflowConfigLoading(true)
    try {
      const workflow = workflows.find((item) => item.name === scanWorkflow)
      if (!workflow) throw new Error("Selected Workflow is unavailable")
      const profile = await loadScanWorkflowProfile(scanWorkflow)
      const draft = adaptWorkflowProfile(profile, workflow)
      const nextConfiguration = serializeWorkflowProfileDraft(draft)
      formValuesCacheRef.current = {}
      setConfiguration(nextConfiguration)
      setWorkflowProfileDraft(draft)
      setIsConfigEdited(false)
      setIsYamlValid(true)
      setReplacementConfiguration(nextConfiguration)
    } catch {
      // Keep the current draft when an explicit Profile reset cannot initialize.
    } finally {
      setIsWorkflowConfigLoading(false)
    }
  }, [loadScanWorkflowProfile, scanWorkflow, workflows])

  React.useEffect(() => {
    if (open) return
    profileBootstrapRequestRef.current += 1
    formValuesCacheRef.current = {}
    setWorkflowProfileDraft(null)
    setProfileBootstrapWorkflowName(null)
    setIsWorkflowProfileLoading(false)
  }, [open])

  const handleTargetSelect = React.useCallback((targetId: number) => {
    setSelectedTargetId((prev) => (prev === targetId ? null : targetId))
  }, [])

  const isConfigurationChanged = replacementConfiguration !== null || isConfigEdited
  const isConfigurationSaveBlocked = isConfigurationChanged && (
    engineCatalog.isLoading
    || engineCatalog.isError
    || !selectedWorkflowWithEngines
    || !configuration.trim()
    || !isYamlValid
  )

  const handleSubmit = React.useCallback(() => {
    if (!scheduledScan) return
    if (isWorkflowConfigLoading) return

    if (isConfigurationChanged) {
      if (isConfigurationSaveBlocked) {
        toastFeedback.error(t("toast.configConflict"))
        return
      }
      if (configValidationRef.current?.validateAndReveal() === false) {
        toastFeedback.error(t("toast.configConflict"))
        return
      }
    }

    if (!displayName.trim()) {
      toastFeedback.error(t("form.taskNameRequired"))
      return
    }
    if (!scanWorkflow) {
      toastFeedback.error(t("form.scanWorkflowRequired"))
      return
    }
    if (scheduledScan.scanMode === "target" && !selectedTargetId) {
      toastFeedback.error(t("toast.selectTarget"))
      return
    }
    if (!timeZone.trim()) {
      toastFeedback.error(t("form.timeZoneRequired"))
      return
    }
    if (!isIanaTimeZoneValid(timeZone)) {
      toastFeedback.error(t("form.timeZoneInvalid"))
      return
    }
    if (!isCronExpressionValid(cronExpression)) {
      toastFeedback.error(t("form.cronRequired"))
      return
    }
    const request: UpdateScheduledScanRequest = {
      displayName: displayName.trim(),
      scanWorkflow: scanWorkflow,
      cronExpression: cronExpression.trim(),
      agentId: selectedAgentID,
    }

    if (timeZone.trim() !== scheduledScan.timeZone) {
      request.timeZone = timeZone.trim()
    }

    if (isConfigurationChanged) {
      request.configuration = configuration
    }

    if (inputSource !== scheduledScan.inputSource) {
      request.inputSource = inputSource
    }

    if (scheduledScan.scanMode === "target" && selectedTargetId) {
      request.targetId = selectedTargetId
    }

    updateScheduledScan(
      { id: scheduledScan.id, data: request },
      {
        onSuccess: () => {
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
      }
    )
  }, [configuration, cronExpression, displayName, inputSource, isConfigurationChanged, isConfigurationSaveBlocked, isWorkflowConfigLoading, onOpenChange, onSuccess, scanWorkflow, scheduledScan, selectedAgentID, selectedTargetId, t, timeZone, updateScheduledScan])

  const getCronDescription = React.useCallback((cron: string): string => {
    try {
      if (cron.trim().split(/\s+/).length !== 5) return t("form.invalidExpression")
      return cronstrue.toString(cron, { locale: locale === "zh" ? "zh_CN" : "en" })
    } catch {
      return t("form.invalidExpression")
    }
  }, [locale, t])

  const getNextExecutions = React.useCallback((cron: string, zone: string, count: number = 3): string[] => (
    getNextCronExecutions(cron, zone, new Date(), count).map((next) => (
      next.toLocaleString(locale === "zh" ? "zh-CN" : "en-US", { timeZone: zone })
    ))
  ), [locale])

  return {
    cronPresets,
    displayName,
    setDisplayName,
    scanWorkflow,
    selectedTargetId,
    selectedAgentID,
    inputSource,
    timeZone,
    cronExpression,
    setCronExpression,
    configuration,
    isConfigEdited,
    isYamlValid,
    isConfigurationSaveBlocked,
    selectedWorkflows: selectedWorkflow ? [selectedWorkflow] : [],
    engineCatalogDetails: engineCatalog.data,
    isEngineCatalogLoading: engineCatalog.isLoading,
    isEngineCatalogError: engineCatalog.isError,
    configValidationRef,
    formValuesCacheRef,
    workflowProfileDraft,
    isWorkflowProfileLoading,
    targets,
    workflows,
    isPending,
    isLoadingWorkflows,
    isWorkflowsError,
    isWorkflowConfigLoading,
    handleWorkflowNamesChange,
    handleConfigSync,
    handleManualConfigChange,
    handleResetWorkflowConfig,
    handleYamlValidationChange,
    handleTargetSelect,
    setSelectedAgentID,
    setInputSource,
    setTimeZone,
    handleSubmit,
    getCronDescription,
    getNextExecutions,
  }
}
