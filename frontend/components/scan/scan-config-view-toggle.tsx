"use client"

import React, { useCallback, useEffect, useState } from "react"
import * as yaml from "js-yaml"
import { useTranslations } from "next-intl"
import { FileCode, RefreshCw } from "@/components/icons"

import { cn } from "@/lib/utils"
import { textRole } from "@/lib/typography"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Switch } from "@/components/ui/switch"
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
} from "./engine-config-form"
import { parseWorkflowConfigurationDraftStrict } from "@/lib/workflow-config"
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
  disabled?: boolean
  isConfigEdited?: boolean
  className?: string
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
  disabled = false,
  isConfigEdited = false,
  className,
}: ScanConfigViewToggleProps, ref) {
  const t = useTranslations("scan.initiate")
  const [viewMode, setViewMode] = useState<ConfigViewMode>("form")
  const wordlistCatalog = useCompleteWordlistCatalogState()
  const [fieldErrors, setFieldErrors] = useState<Map<string, ConfigResourceFieldError>>(
    () => new Map()
  )
  const [expandedStepIds, setExpandedStepIds] = useState<Set<string>>(() => new Set())
  const [pendingFocusFieldId, setPendingFocusFieldId] = useState<string | null>(null)

  const [formValues, setFormValues] = useState<EngineConfigFormValues>(() =>
    safelyInitFormValues(workflow, configuration)
  )
  const formValuesRef = React.useRef(formValues)
  formValuesRef.current = formValues

  const lastSerializedRef = React.useRef<string>("")

  useEffect(() => {
    try {
      const fresh = initFormValuesFromWorkflow(workflow, configuration, formValuesRef.current)
      setFormValues(fresh)
      onValidationChange?.(true)
    } catch {
      // A Profile/YAML that is not complete must not be repaired from catalog defaults.
      setFormValues({})
      onValidationChange?.(false)
    }
    lastSerializedRef.current = ""
  }, [configuration, onValidationChange, workflow])

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
    setFormValues(reconciled)
    const applySyncedConfig = onSync ?? onChange
    applySyncedConfig(yamlStr)
  }, [onChange, onSync, serializeFormToYaml, workflow, wordlistCatalog.status, wordlistCatalog.wordlists])

  const handleFormChange = useCallback(
    (values: EngineConfigFormValues) => {
      setFormValues(values)
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
    [onChange, serializeFormToYaml, workflow]
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
    [configuration, formValues, onChange, onSync, serializeFormToYaml]
  )

  useEffect(() => {
    if (viewMode !== "form") return
    if (!configuration || configuration === lastSerializedRef.current) return

    try {
      const parsed = parseWorkflowConfigurationDraftStrict(configuration)
      const rebuilt = initFormValuesFromWorkflow(workflow, parsed, formValuesRef.current)
      setFormValues(rebuilt)
      lastSerializedRef.current = configuration
      onValidationChange?.(true)
    } catch {
      onValidationChange?.(false)
    }
  }, [configuration, onValidationChange, viewMode, workflow])

  const isFormMode = viewMode === "form"

  const handleEditorValidationChange = useCallback(
    (syntaxValid: boolean) => {
      if (!syntaxValid) {
        onValidationChange?.(false)
        return
      }
      try {
        // The YAML editor owns text syntax only. Re-run the same Profile/schema
        // adapter here so YAML cannot bypass exact Step and Engine validation.
        initFormValuesFromWorkflow(workflow, configuration, formValuesRef.current)
        onValidationChange?.(true)
      } catch {
        onValidationChange?.(false)
      }
    },
    [configuration, onValidationChange, workflow]
  )

  React.useImperativeHandle(ref, () => ({
    validateAndReveal: () => {
      let currentValues: EngineConfigFormValues
      try {
        const parsed = parseWorkflowConfigurationDraftStrict(configuration)
        currentValues = initFormValuesFromWorkflow(
          workflow,
          parsed,
          formValuesRef.current
        )
        onValidationChange?.(true)
      } catch {
        onValidationChange?.(false)
        return false
      }

      const errors = validateRequiredConfigResources(workflow, currentValues)
      if (errors.length === 0) {
        setFieldErrors(new Map())
        return true
      }

      const nextErrors = new Map(errors.map((error) => [error.key, error]))
      const firstError = errors[0]
      setFormValues(currentValues)
      setFieldErrors(nextErrors)
      setExpandedStepIds((current) => new Set(current).add(firstError.stepId))
      setViewMode("form")
      setPendingFocusFieldId(firstError.fieldId)
      return false
    },
  }), [configuration, onValidationChange, workflow])

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
        <div className="radius-surface flex flex-wrap items-center justify-between gap-3 border bg-muted/20 px-4 py-3">
          <div className="flex min-w-0 items-center gap-3">
          <span className="flex size-7 shrink-0 items-center justify-center text-muted-foreground">
            <FileCode className="size-7" />
          </span>
          <div className="min-w-0 space-y-1">
            <p className={textRole.sectionTitle}>{t("advancedYamlTitle")}</p>
            <p className={textRole.helperText}>{t("advancedYamlHelper")}</p>
          </div>
          </div>

          <div className="flex items-center gap-2">
          {isFormMode && onReset ? (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={handleResetToDefaults}
              disabled={disabled}
              className="gap-1.5"
            >
              <RefreshCw className="size-3" />
              {t("resetDefaults")}
            </Button>
          ) : null}
          <Switch
            checked={!isFormMode}
            onCheckedChange={handleViewModeChange}
            disabled={disabled}
            aria-label={t("advancedYamlTitle")}
          />
          </div>
        </div>

        <div className="flex min-h-0 flex-1 flex-col gap-3">
          <div className="flex items-center justify-between gap-3">
          <h3 className={textRole.sectionTitle}>{t("steps.engineConfig")}</h3>
          {isConfigEdited ? (
            <span className="inline-flex shrink-0 items-center gap-1 text-[11px] text-warning">
              <span className="size-1.5 rounded-full bg-warning" />
              {t("configEdited")}
            </span>
          ) : null}
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
              onChange={handleFormChange}
            />
            ) : (
            <ScanConfigEditor
              configuration={configuration}
              onChange={onChange}
              onValidationChange={handleEditorValidationChange}
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
      </div>
    </ScrollArea>
  )
})

function safelyInitFormValues(
  workflow: ScanWorkflowWithEngines,
  configuration: string,
): EngineConfigFormValues {
  try {
    return initFormValuesFromWorkflow(workflow, configuration)
  } catch {
    return {}
  }
}
