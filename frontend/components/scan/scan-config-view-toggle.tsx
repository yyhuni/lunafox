"use client"

import React, { useCallback, useEffect, useState } from "react"
import * as yaml from "js-yaml"
import { useTranslations } from "next-intl"
import { RefreshCw, Zap } from "@/components/icons"

import { cn } from "@/lib/utils"
import { textRole } from "@/lib/typography"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useCompleteWordlistCatalogState } from "@/hooks/use-wordlists"
import {
  configResourceFieldKey,
  reconcileConfirmedWordlistCatalog,
  validateRequiredConfigResources,
  type ConfigResourceFieldError,
  type ConfigResourceFieldLocation,
} from "@/lib/engine-config-resource-validation"

import { ScanConfigEditor } from "./scan-config-editor"
import {
  EngineConfigForm,
  initFormValuesFromWorkflow,
  serializeFormValuesToConfig,
  type EngineParamControlRenderer,
} from "./engine-config-form"
import {
  parseWorkflowConfigurationDraftStrict,
  type WorkflowProfileDraft,
} from "@/lib/workflow-config"
import type {
  EngineConfigFormValues,
  ScanWorkflowWithEngines,
} from "@/types/engine-config.types"

type ConfigViewMode = "form" | "yaml"

interface ScanConfigViewToggleProps {
  workflow: ScanWorkflowWithEngines
  configuration: string
  onChange: (value: string) => void
  onSync?: (value: string) => void
  onReset?: () => void
  onValidationChange?: (isValid: boolean) => void
  selectedScanWorkflows?: Array<{ name: string; configuration?: unknown }>
  formValuesCacheRef?: React.MutableRefObject<EngineConfigFormValues>
  workflowProfileDraft?: WorkflowProfileDraft | null
  disabled?: boolean
  isConfigEdited?: boolean
  className?: string
  renderParamControl?: EngineParamControlRenderer
  defaultExpandedStepIds?: ReadonlySet<string>
  workflowSummary?: string
  configuredEngineCount?: number
  configuredStageCount?: number
}

export type ScanConfigValidationHandle = {
  validateAndReveal: () => boolean
}

export const ScanConfigViewToggle = React.forwardRef<
  ScanConfigValidationHandle,
  ScanConfigViewToggleProps
>(function ScanConfigViewToggle({
  workflow,
  configuration,
  onChange,
  onSync,
  onReset,
  onValidationChange,
  selectedScanWorkflows = [],
  formValuesCacheRef,
  workflowProfileDraft = null,
  disabled = false,
  isConfigEdited = false,
  className,
  renderParamControl,
  defaultExpandedStepIds,
  workflowSummary,
  configuredEngineCount,
  configuredStageCount,
}: ScanConfigViewToggleProps, ref) {
  const t = useTranslations("scan.initiate")
  const [viewMode, setViewMode] = useState<ConfigViewMode>("form")
  const wordlistCatalog = useCompleteWordlistCatalogState()
  const [fieldErrors, setFieldErrors] = useState<Map<string, ConfigResourceFieldError>>(
    () => new Map()
  )
  const [expandedStepIds, setExpandedStepIds] = useState<Set<string>>(
    () => new Set(defaultExpandedStepIds ?? [])
  )
  const [pendingFocusFieldId, setPendingFocusFieldId] = useState<string | null>(null)
  const [schemaValidationError, setSchemaValidationError] = useState<string | null>(null)

  const [formValues, setFormValues] = useState<EngineConfigFormValues>(() =>
    safelyInitFormValues(workflow, configuration, formValuesCacheRef?.current, workflowProfileDraft)
  )
  const formValuesRef = React.useRef(formValues)
  formValuesRef.current = formValues

  const cacheFormValues = useCallback((values: EngineConfigFormValues) => {
    formValuesRef.current = values
    if (formValuesCacheRef) {
      formValuesCacheRef.current = values
    }
  }, [formValuesCacheRef])

  const replaceFormValues = useCallback((values: EngineConfigFormValues) => {
    cacheFormValues(values)
    setFormValues(values)
  }, [cacheFormValues])

  const getCachedFormValues = useCallback(
    () => formValuesCacheRef?.current ?? formValuesRef.current,
    [formValuesCacheRef]
  )

  const schemaValidationErrorFallback = t("configSchemaErrorFallback")
  const reportSchemaValidationFailure = useCallback((error: unknown) => {
    const message = error instanceof Error && error.message.trim().length > 0
      ? error.message
      : schemaValidationErrorFallback
    setSchemaValidationError(message)
    onValidationChange?.(false)
  }, [onValidationChange, schemaValidationErrorFallback])

  const lastSerializedRef = React.useRef<string>("")

  useEffect(() => {
    try {
      const fresh = initFormValuesFromWorkflow(
        workflow,
        configuration,
        getCachedFormValues(),
        workflowProfileDraft,
      )
      replaceFormValues(fresh)
      setSchemaValidationError(null)
      onValidationChange?.(true)
    } catch (error) {
      // A Profile/YAML that is not complete must not be repaired from catalog defaults.
      setFormValues({})
      reportSchemaValidationFailure(error)
    }
    lastSerializedRef.current = ""
  }, [configuration, getCachedFormValues, onValidationChange, replaceFormValues, reportSchemaValidationFailure, workflow, workflowProfileDraft])

  const serializeFormToYaml = useCallback(
    (values: EngineConfigFormValues): string => {
      if (Object.keys(values).length === 0) return ""
      const configObj = serializeFormValuesToConfig(values)
      if (Object.keys(configObj).length === 0) return ""
      return yaml.dump(configObj, { lineWidth: -1, noRefs: true }).trim()
    },
    []
  )

  useEffect(() => {
    if (wordlistCatalog.status !== "complete") return
    const reconciled = reconcileConfirmedWordlistCatalog(
      workflow,
      formValuesRef.current,
      wordlistCatalog.wordlists
    )
    if (reconciled === formValuesRef.current) return

    const yamlStr = serializeFormToYaml(reconciled)
    if (yamlStr === lastSerializedRef.current) return

    // React may invoke state updaters during render, so parent sync must stay outside them.
    lastSerializedRef.current = yamlStr
    replaceFormValues(reconciled)
    const applySyncedConfig = onSync ?? onChange
    applySyncedConfig(yamlStr)
  }, [onChange, onSync, replaceFormValues, serializeFormToYaml, workflow, wordlistCatalog.status, wordlistCatalog.wordlists])

  const handleFormChange = useCallback(
    (values: EngineConfigFormValues) => {
      replaceFormValues(values)
      setFieldErrors((current) => {
        if (current.size === 0) return current
        const remainingRequired = new Set(
          validateRequiredConfigResources(workflow, values).map((error) => error.key)
        )
        const next = new Map(
          Array.from(current.entries()).filter(([key]) => remainingRequired.has(key))
        )
        return next.size === current.size ? current : next
      })
      const yamlStr = serializeFormToYaml(values)
      if (yamlStr !== lastSerializedRef.current) {
        lastSerializedRef.current = yamlStr
        onChange(yamlStr)
      }
    },
    [onChange, replaceFormValues, serializeFormToYaml, workflow]
  )

  const handleExpandedStepChange = useCallback((stepId: string, open: boolean) => {
    setExpandedStepIds((current) => {
      const next = new Set(current)
      if (open) next.add(stepId)
      else next.delete(stepId)
      return next
    })
  }, [])

  const handleFieldRepaired = useCallback((location: ConfigResourceFieldLocation) => {
    const key = configResourceFieldKey(location)
    setFieldErrors((current) => {
      if (!current.has(key)) return current
      const next = new Map(current)
      next.delete(key)
      return next
    })
  }, [])

  const handleResetToDefaults = useCallback(() => {
    if (onReset) {
      onReset()
      return
    }
  }, [onReset])

  const handleViewModeChange = useCallback(
    (checked: boolean) => {
      cacheFormValues(formValues)
      if (checked) {
        const yamlStr = serializeFormToYaml(formValues)
        if (yamlStr !== lastSerializedRef.current) {
          lastSerializedRef.current = yamlStr
          if (yamlStr !== configuration) {
            if (yamlStr) {
              const applySyncedConfig = onSync ?? onChange
              applySyncedConfig(yamlStr)
            }
          }
        }
      }
      setViewMode(checked ? "yaml" : "form")
    },
    [cacheFormValues, configuration, formValues, onChange, onSync, serializeFormToYaml]
  )

  useEffect(() => {
    if (viewMode !== "form") return
    if (!configuration || configuration === lastSerializedRef.current) return

    try {
      const parsed = parseWorkflowConfigurationDraftStrict(configuration)
      const rebuilt = initFormValuesFromWorkflow(
        workflow,
        parsed,
        getCachedFormValues(),
        workflowProfileDraft,
      )
      replaceFormValues(rebuilt)
      lastSerializedRef.current = configuration
      setSchemaValidationError(null)
      onValidationChange?.(true)
    } catch (error) {
      reportSchemaValidationFailure(error)
    }
  }, [configuration, getCachedFormValues, onValidationChange, replaceFormValues, reportSchemaValidationFailure, viewMode, workflow, workflowProfileDraft])

  const isFormMode = viewMode === "form"

  const handleEditorValidationChange = useCallback(
    (syntaxValid: boolean) => {
      if (!syntaxValid) {
        setSchemaValidationError(null)
        onValidationChange?.(false)
        return
      }
      try {
        // The YAML editor owns text syntax only. Re-run the same Profile/schema
        // adapter here so YAML cannot bypass exact Step and Engine validation.
        initFormValuesFromWorkflow(
          workflow,
          configuration,
          getCachedFormValues(),
          workflowProfileDraft,
        )
        setSchemaValidationError(null)
        onValidationChange?.(true)
      } catch (error) {
        reportSchemaValidationFailure(error)
      }
    },
    [configuration, getCachedFormValues, onValidationChange, reportSchemaValidationFailure, workflow, workflowProfileDraft]
  )

  React.useImperativeHandle(ref, () => ({
    validateAndReveal: () => {
      let currentValues: EngineConfigFormValues
      try {
        const parsed = parseWorkflowConfigurationDraftStrict(configuration)
        currentValues = initFormValuesFromWorkflow(
          workflow,
          parsed,
          getCachedFormValues(),
          workflowProfileDraft,
        )
        replaceFormValues(currentValues)
        setSchemaValidationError(null)
        onValidationChange?.(true)
      } catch (error) {
        reportSchemaValidationFailure(error)
        return false
      }

      const errors = validateRequiredConfigResources(workflow, currentValues)
      if (errors.length === 0) {
        setFieldErrors(new Map())
        return true
      }

      const nextErrors = new Map(errors.map((error) => [error.key, error]))
      const firstError = errors[0]
      setFieldErrors(nextErrors)
      setExpandedStepIds((current) => new Set(current).add(firstError.stepId))
      setViewMode("form")
      setPendingFocusFieldId(firstError.fieldId)
      return false
    },
  }), [configuration, getCachedFormValues, onValidationChange, replaceFormValues, reportSchemaValidationFailure, workflow, workflowProfileDraft])

  useEffect(() => {
    if (!pendingFocusFieldId || viewMode !== "form") return
    const field = document.getElementById(pendingFocusFieldId)
    if (!field) return
    field.scrollIntoView?.({ block: "center", behavior: "smooth" })
    field.focus({ preventScroll: true })
    setPendingFocusFieldId(null)
  }, [expandedStepIds, pendingFocusFieldId, viewMode])

  return (
    <ScrollArea className={cn("min-h-0 flex-1", className)} contentClassName="!min-w-0 w-full">
      <div className="flex min-h-full flex-col gap-3">
        <div className="flex items-center justify-between gap-3">
          <div className="flex min-w-0 flex-1 items-center gap-2">
            <h3 className={cn("shrink-0", textRole.sectionTitle)}>{t("steps.engineConfig")}</h3>
            {workflowSummary ? (
              <span
                title={workflowSummary}
                className="inline-flex min-w-0 max-w-[130px] sm:max-w-[200px] shrink items-center gap-1 rounded border border-border/60 bg-muted/30 px-1.5 py-0.5 text-[11px] text-muted-foreground"
              >
                <Zap className="size-3 shrink-0" />
                <span className="truncate">{workflowSummary}</span>
              </span>
            ) : null}
            {isConfigEdited ? (
              <span className="inline-flex shrink-0 items-center gap-1 text-[11px] text-warning">
                <span className="size-1.5 rounded-full bg-warning" />
                {t("configEdited")}
              </span>
            ) : null}
          </div>

          <div className="flex shrink-0 items-center gap-2">
            {isFormMode && onReset ? (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={handleResetToDefaults}
                disabled={disabled}
                className="h-7 gap-1 px-2 text-xs text-muted-foreground hover:text-foreground"
              >
                <RefreshCw className="size-3" />
                {t("resetDefaults")}
              </Button>
            ) : null}
            <Tabs
              value={viewMode}
              onValueChange={(val) => handleViewModeChange(val === "yaml")}
            >
              <TabsList variant="filter" size="sm" className="h-7 p-[2px]" aria-label={t("advancedYamlTitle")}>
                <TabsTrigger value="form" variant="filter" size="sm" className="h-[calc(100%-1px)] px-2 py-0.5 text-xs">
                  {t("modeForm")}
                </TabsTrigger>
                <TabsTrigger value="yaml" variant="filter" size="sm" className="h-[calc(100%-1px)] px-2 py-0.5 text-xs">
                  {t("modeYaml")}
                </TabsTrigger>
              </TabsList>
            </Tabs>
          </div>
        </div>

        <div className="min-h-0 flex-1">
          {isFormMode ? (
            <EngineConfigForm
              workflow={workflow}
              values={formValues}
              disabled={disabled}
              wordlistCatalog={wordlistCatalog}
              fieldErrors={fieldErrors}
              expandedStepIds={expandedStepIds}
              onExpandedStepChange={handleExpandedStepChange}
              onFieldRepaired={handleFieldRepaired}
              renderParamControl={renderParamControl}
              onChange={handleFormChange}
            />
          ) : (
            <ScanConfigEditor
              configuration={configuration}
              onChange={onChange}
              onValidationChange={handleEditorValidationChange}
              validationError={schemaValidationError}
              selectedScanWorkflows={selectedScanWorkflows as never}
              isConfigEdited={isConfigEdited}
              disabled={disabled}
              showCapabilities={false}
              showLabel={false}
              className="h-full"
            />
          )}
        </div>
      </div>
    </ScrollArea>
  )
})

function safelyInitFormValues(
  workflow: ScanWorkflowWithEngines,
  configuration: string,
  previousValues?: EngineConfigFormValues,
  workflowProfileDraft?: WorkflowProfileDraft | null,
): EngineConfigFormValues {
  try {
    return initFormValuesFromWorkflow(workflow, configuration, previousValues, workflowProfileDraft)
  } catch {
    return {}
  }
}
