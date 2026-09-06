import React from "react"
import {
  AlertTriangle,
  Box,
  CheckCircle2,
  Clock,
  ChevronLeft,
  ChevronRight,
  Play,
  Zap,
  semanticIcons,
} from "@/components/icons"

import { Button } from "@/components/ui/button"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { CommandGroup, CommandItem } from "@/components/ui/command"
import {
  AlertDialog,
  AlertDialogClose, AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Spinner } from "@/components/shared/loading/spinner"
import {
  ScanConfigViewToggle,
  type ScanConfigValidationHandle,
} from "./scan-config-view-toggle"
import { ScanSearchablePicker } from "./scan-searchable-picker"
import { buildWorkflowWithEngineCatalog } from "@/lib/engine-catalog"
import { cn } from "@/lib/utils"
import { textRole } from "@/lib/typography"
import type { InitiateScanSelectMode } from "@/lib/initiate-scan-helpers"
import { parseWorkflowConfigurationDraftStrict } from "@/lib/workflow-config"
import type { Locale } from "@/i18n/config"
import type { ScanWorkflowWithEngines } from "@/types/engine-config.types"
import type { ScanWorkflow } from "@/types/scan-workflow.types"
import type { EngineCatalogDetail } from "@/types/engine-catalog.types"
import type { ScanInputSource } from "@/types/scan.types"
import { ScanAgentSelector } from "@/components/scan/agent-selector"
import { ScanInputSourceSelector } from "@/components/scan/scan-input-source-selector"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

type StepDescriptor = {
  id: number
  title: string
}

const VulnerabilityIcon = semanticIcons.concept.vulnerability

interface InitiateScanStepHeaderProps {
  steps: StepDescriptor[]
  currentStep: number
  progress: number
}

export function getScanStepProgress(currentStep: number, totalSteps: number): number {
  if (!Number.isFinite(currentStep) || !Number.isFinite(totalSteps) || totalSteps <= 0) return 0

  const boundedStep = Math.min(Math.max(currentStep, 0), totalSteps)
  return Math.round((boundedStep / totalSteps) * 100)
}

export function InitiateScanStepHeader({ steps, currentStep, progress }: InitiateScanStepHeaderProps) {
  const progressValue = Math.min(100, Math.max(0, progress))
  const currentTitle = steps.find((step) => step.id === currentStep)?.title

  return (
    <div
      aria-label={currentTitle}
      aria-valuemax={100}
      aria-valuemin={0}
      aria-valuenow={progressValue}
      className="relative h-3"
      role="progressbar"
    >
      <div className="absolute inset-x-1.5 top-1/2 h-0.5 -translate-y-1/2 bg-border/70">
        <div className="h-full bg-interaction-accent transition-[width]" style={{ width: `${progressValue}%` }} />
        <span
          aria-hidden="true"
          className="absolute top-1/2 z-10 size-3 -translate-x-1/2 -translate-y-1/2 rounded-full bg-interaction-accent transition-[left]"
          style={{ left: `${progressValue}%` }}
        />
      </div>
    </div>
  )
}

function getWorkflowIcon(workflowName: string, index: number): React.ReactNode {
  const lowerName = workflowName.toLowerCase()
  if (lowerName.includes("快速") || lowerName.includes("quick")) return <Zap className="size-6" />
  if (lowerName.includes("web")) return <semanticIcons.concept.website className="size-6" />
  if (lowerName.includes("内网") || lowerName.includes("network")) return <semanticIcons.concept.scan className="size-6" />
  if (lowerName.includes("漏洞") || lowerName.includes("vuln")) return <VulnerabilityIcon className="size-6" />
  if (lowerName.includes("资产") || lowerName.includes("asset")) return <semanticIcons.concept.asset className="size-6" />
  if (lowerName.includes("定时") || lowerName.includes("monitor")) return <Clock className="size-6" />

  const icons = [
    semanticIcons.concept.workflow,
    semanticIcons.concept.engine,
    semanticIcons.concept.target,
    semanticIcons.concept.website,
    semanticIcons.concept.scan,
  ]
  const Icon = icons[index % icons.length]
  return <Icon className="size-6" />
}

function countWorkflowEngines(workflow: ScanWorkflowWithEngines): number {
  return workflow.stages.reduce((sum, stage) => sum + stage.steps.filter((step) => step.engine).length, 0)
}

export function countConfiguredWorkflowEngines(configuration: string): number {
  let parsed: Record<string, unknown>
  try {
    parsed = parseWorkflowConfigurationDraftStrict(configuration)
  } catch {
    return 0
  }
  if (!parsed.steps || typeof parsed.steps !== "object" || Array.isArray(parsed.steps)) return 0
  return Object.values(parsed.steps).filter((step) => {
    if (!step || typeof step !== "object" || Array.isArray(step)) return false
    return (step as Record<string, unknown>).enabled !== false
  }).length
}

export function countWorkflowConfiguredEngines(workflows: ScanWorkflow[]): number {
  const workflow = workflows[0]
  if (!workflow?.steps?.length) return 0
  return workflow.steps.length
}

function InitiateScanPanelState({
  icon,
  title,
  description,
}: {
  icon: React.ReactNode
  title: string
  description?: string
}) {
  return (
    <div className="radius-surface flex min-h-32 items-center justify-center border border-dashed bg-muted/20 p-6 text-center">
      <div className="grid justify-items-center gap-2">
        <span className="radius-round flex size-10 items-center justify-center border bg-background text-muted-foreground">
          {icon}
        </span>
        <p className={textRole.navLabel}>{title}</p>
        {description ? <p className={textRole.helperText}>{description}</p> : null}
      </div>
    </div>
  )
}

interface InitiateScanWorkflowSelectionProps {
  t: TranslationFn
  selectedWorkflowNames: string[]
  workflows?: ScanWorkflow[]
  isLoadingWorkflows: boolean
  isWorkflowsError: boolean
  isSubmitting: boolean
  onWorkflowNamesChange: (workflowNames: string[]) => void
}

export function InitiateScanWorkflowSelection({
  t,
  selectedWorkflowNames,
  workflows,
  isLoadingWorkflows,
  isWorkflowsError,
  isSubmitting,
  onWorkflowNamesChange,
}: InitiateScanWorkflowSelectionProps) {
  const selectedWorkflowName = selectedWorkflowNames[0] ?? ""
  const selectedWorkflowIndex = workflows?.findIndex((workflow) => workflow.name === selectedWorkflowName) ?? -1
  const selectedWorkflow = selectedWorkflowIndex >= 0 ? workflows?.[selectedWorkflowIndex] : undefined
  const selectedWorkflowDisplayName = selectedWorkflow?.displayName || selectedWorkflow?.name || t("selectWorkflow")

  const selectWorkflow = (workflow: ScanWorkflow) => {
    if (!workflow.isExecutable) return
    onWorkflowNamesChange([workflow.name])
  }

  return (
    <div className="grid gap-4">
      <div className="grid gap-4">
        <div className="space-y-1">
          <p className={textRole.sectionTitle}>{t("workflowListTitle")}</p>
          <p className={textRole.helperText}>{t("workflowListHint")}</p>
        </div>
        {isLoadingWorkflows ? (
          <InitiateScanPanelState
            icon={<Spinner />}
            title={t("loading")}
          />
        ) : isWorkflowsError ? (
          <InitiateScanPanelState
            icon={<AlertTriangle className="size-5 text-warning" />}
            title={t("loadFailed")}
          />
        ) : workflows && workflows.length > 0 ? (
          <ScanSearchablePicker
            ariaLabel={t("workflowListTitle")}
            disabled={isSubmitting}
            trigger={
              <span className="flex min-w-0 flex-1 items-center gap-2 overflow-hidden">
                <span className="flex size-8 shrink-0 items-center justify-center text-muted-foreground">
                  {getWorkflowIcon(selectedWorkflowDisplayName, Math.max(selectedWorkflowIndex, 0))}
                </span>
                <span className="min-w-0 flex-1 overflow-hidden">
                  <span className={cn("block truncate", textRole.navLabel)}>{selectedWorkflowDisplayName}</span>
                  {selectedWorkflow?.description ? (
                    <span className={cn("block truncate", textRole.helperText)}>{selectedWorkflow.description}</span>
                  ) : null}
                </span>
              </span>
            }
            searchPlaceholder={t("workflowSearchPlaceholder")}
            emptyLabel={t("workflowSearchEmpty")}
          >
            <CommandGroup>
              {workflows.map((workflow, workflowIndex) => {
                const workflowDisplayName = workflow.displayName || workflow.name
                const isSelected = selectedWorkflowName === workflow.name
                const isUnavailable = !workflow.isExecutable
                return (
                  <CommandItem
                    key={workflow.name}
                    value={workflow.name}
                    keywords={[workflowDisplayName, workflow.description ?? ""]}
                    disabled={isUnavailable}
                    onSelect={() => selectWorkflow(workflow)}
                    className="min-h-12 gap-3 py-2"
                  >
                    <span className="flex size-8 shrink-0 items-center justify-center text-muted-foreground">
                      {getWorkflowIcon(workflowDisplayName, workflowIndex)}
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className={cn("block truncate", textRole.navLabel)}>{workflowDisplayName}</span>
                      {workflow.description ? <span className={cn("block truncate", textRole.helperText)}>{workflow.description}</span> : null}
                    </span>
                    {isSelected ? <CheckCircle2 className="size-4 shrink-0 text-primary" /> : null}
                    {isUnavailable ? <span className="shrink-0 text-xs text-muted-foreground">{t("workflowUnavailable")}</span> : null}
                  </CommandItem>
                )
              })}
            </CommandGroup>
          </ScanSearchablePicker>
        ) : (
          <InitiateScanPanelState
            icon={<semanticIcons.concept.workflow className="size-5" />}
            title={t("noWorkflows")}
          />
        )}
      </div>
    </div>
  )
}

interface InitiateScanExecutionOptionsProps {
  t: TranslationFn
  inputSource: ScanInputSource
  selectedAgentID: number | null
  isSubmitting: boolean
  onInputSourceChange: (value: ScanInputSource) => void
  onAgentChange: (value: number | null) => void
}

export function InitiateScanExecutionOptions({
  t,
  inputSource,
  selectedAgentID,
  isSubmitting,
  onInputSourceChange,
  onAgentChange,
}: InitiateScanExecutionOptionsProps) {
  return (
    <Collapsible defaultOpen={false} className="mt-5 border-t border-border/60 pt-4">
      <CollapsibleTrigger
        render={(
          <Button
            type="button"
            variant="ghost"
            className="group h-auto min-w-0 justify-start px-0 py-0 text-left hover:bg-transparent hover:text-foreground dark:hover:bg-transparent dark:hover:text-foreground data-[panel-open]:[&>svg]:rotate-90"
          />
        )}
      >
        <ChevronRight className="size-4 shrink-0 text-muted-foreground transition-[color,transform] duration-200 motion-reduce:transition-none group-hover:text-foreground" />
        <span className={cn(textRole.bodyStrong, "min-w-0 text-muted-foreground transition-colors group-hover:text-foreground")}>
          {t("executionOptions.title")}
        </span>
      </CollapsibleTrigger>
      <CollapsibleContent className="space-y-5 pt-4">
        <ScanInputSourceSelector
          id="initiate-scan-input-source"
          value={inputSource}
          onValueChange={onInputSourceChange}
          disabled={isSubmitting}
        />
        <ScanAgentSelector value={selectedAgentID} onChange={onAgentChange} disabled={isSubmitting} />
      </CollapsibleContent>
    </Collapsible>
  )
}

interface InitiateScanConfigStepProps {
  t: TranslationFn
  configuration: string
  selectedWorkflows: ScanWorkflow[]
  isConfigEdited: boolean
  isYamlValid: boolean
  hasConfig: boolean
  selectMode: InitiateScanSelectMode
  selectedWorkflowNames: string[]
  locale?: Locale
  engineCatalogDetails?: EngineCatalogDetail[]
  isEngineCatalogLoading: boolean
  isEngineCatalogError: boolean
  isSubmitting: boolean
  onConfigSync: (value: string) => void
  onConfigChange: (value: string) => void
  onResetConfig: () => void
  onYamlValidationChange: (isValid: boolean) => void
  configValidationRef?: React.Ref<ScanConfigValidationHandle>
}

export function InitiateScanConfigStep({
  t,
  configuration,
  selectedWorkflows,
  isConfigEdited,
  isYamlValid,
  hasConfig,
  selectMode,
  selectedWorkflowNames,
  locale,
  engineCatalogDetails,
  isEngineCatalogLoading,
  isEngineCatalogError,
  isSubmitting,
  onConfigSync,
  onConfigChange,
  onResetConfig,
  onYamlValidationChange,
  configValidationRef,
}: InitiateScanConfigStepProps) {
  const selectionSummary = selectMode === "custom" && selectedWorkflowNames.length > 0
      ? selectedWorkflowNames.join("、")
      : t("validation.workflowsMissing")
  const selectedWorkflow = selectedWorkflows[0]
  const workflowResult = React.useMemo(() => {
    if (!selectedWorkflow?.steps?.length || !engineCatalogDetails || !locale) return null
    try {
      return { workflow: buildWorkflowWithEngineCatalog(selectedWorkflow, engineCatalogDetails, locale), error: null }
    } catch (error) {
      return { workflow: null, error }
    }
  }, [engineCatalogDetails, locale, selectedWorkflow])
  if (isEngineCatalogLoading) {
    return <InitiateScanPanelState icon={<Spinner />} title={t("loading")} />
  }
  if (isEngineCatalogError || !workflowResult?.workflow || workflowResult.error) {
    return <InitiateScanPanelState icon={<AlertTriangle className="size-5 text-warning" />} title={t("loadFailed")} />
  }
  const workflowModel = workflowResult.workflow
  const configuredEngineCount = configuration.trim()
    ? countConfiguredWorkflowEngines(configuration)
    : countWorkflowEngines(workflowModel)
  const configuredStageCount = workflowModel.stages.length

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex min-w-0 flex-wrap items-center gap-x-5 gap-y-2 text-muted-foreground">
          <div className="flex min-w-0 items-center gap-1.5">
            <Zap className="size-4 shrink-0" />
            <span className={cn("min-w-0 truncate", textRole.metadataValueStrong)}>{selectionSummary}</span>
          </div>
        </div>
        <div className="flex shrink-0 flex-wrap items-center gap-x-4 gap-y-1.5 text-muted-foreground">
          <div className="flex items-center gap-1">
            <Box className="size-3.5 shrink-0" />
            <span className={textRole.helperText}>{t("engineConfiguredSummary", { count: configuredEngineCount })}</span>
          </div>
          <div className="flex items-center gap-1">
            <Clock className="size-3.5 shrink-0" />
            <span className={textRole.helperText}>{t("stageCount", { count: configuredStageCount })}</span>
          </div>
          {!isYamlValid || !hasConfig ? <span className={cn("text-warning", textRole.helperText)}>{t("validation.yamlError")}</span> : null}
        </div>
      </div>

      <div className="flex min-h-0 flex-1 flex-col">
        <ScanConfigViewToggle
          ref={configValidationRef}
          workflow={workflowModel}
          configuration={configuration}
          onSync={onConfigSync}
          onChange={onConfigChange}
          onReset={onResetConfig}
          onValidationChange={onYamlValidationChange}
          selectedScanWorkflows={selectedWorkflows}
          disabled={isSubmitting}
          isConfigEdited={isConfigEdited}
        />
      </div>
    </div>
  )
}

interface InitiateScanFooterProps {
  t: TranslationFn
  currentStep: number
  selectMode: InitiateScanSelectMode
  selectedWorkflowNames: string[]
  selectedWorkflowDisplayName?: string
  canProceedToReview: boolean
  canStart: boolean
  hasNoEnabledSteps: boolean
  configuredEngineCount: number
  isSubmitting: boolean
  showBackButton?: boolean
  onBack: () => void
  onNext: () => void
  onStart: () => void
}

export function InitiateScanFooter({
  t,
  currentStep,
  selectMode,
  selectedWorkflowNames,
  selectedWorkflowDisplayName,
  canProceedToReview,
  canStart,
  hasNoEnabledSteps,
  configuredEngineCount,
  isSubmitting,
  showBackButton,
  onBack,
  onNext,
  onStart,
}: InitiateScanFooterProps) {
  const shouldShowBackButton = showBackButton ?? currentStep > 1
  const statusText = currentStep === 1
    ? selectMode === "custom" && selectedWorkflowNames[0]
        ? selectedWorkflowDisplayName || selectedWorkflowNames[0]
        : t("validation.workflowsMissing")
    : canStart
      ? t("configuredEngineStatus", { count: configuredEngineCount })
      : hasNoEnabledSteps
        ? t("validation.noEnabledSteps")
      : t("validation.yamlError")
  const isReady = currentStep === 1 ? canProceedToReview : canStart

  return (
    <div className="flex shrink-0 flex-col gap-3 md:flex-row md:items-center md:justify-between">
      <div className="flex min-h-10 min-w-0 items-start gap-2">
        {isReady ? (
          <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-success" />
        ) : (
          <AlertTriangle className="mt-0.5 size-4 shrink-0 text-warning" />
        )}
        <div className="min-w-0">
          <p className={cn("truncate", isReady ? "text-primary" : undefined, textRole.navLabel)}>
            {statusText}
          </p>
          <p className={textRole.helperText}>
            {currentStep === 1 ? t("steps.selectWorkflow") : t("steps.engineConfig")}
          </p>
        </div>
      </div>

      <div className="flex items-center justify-between gap-3 md:justify-end">
        <div className="flex gap-2">
          {shouldShowBackButton && (
            <Button type="button" variant="outline" size="sm" onClick={onBack} disabled={isSubmitting}>
              <ChevronLeft className="size-4" />
              {t("back")}
            </Button>
          )}
        </div>
        {currentStep === 1 ? (
          <Button type="button" size="sm" onClick={onNext} disabled={!canProceedToReview || isSubmitting}>
            {t("next")}
            <ChevronRight className="size-4" />
          </Button>
        ) : (
          <Button
            type="button"
            size="sm"
            onClick={onStart}
            disabled={!canStart || isSubmitting}
            loading={isSubmitting}
            loadingLabel={t("initiating")}
          >
            <>
              <Play className="mr-1.5 size-4" />
              {t("startScan")}
            </>
          </Button>
        )}
      </div>
    </div>
  )
}

interface InitiateScanOverwriteDialogProps {
  t: TranslationFn
  open: boolean
  onOpenChange: (open: boolean) => void
  onCancel: () => void
  onConfirm: () => void
}

export function InitiateScanOverwriteDialog({
  t,
  open,
  onOpenChange,
  onCancel,
  onConfirm,
}: InitiateScanOverwriteDialogProps) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("overwriteConfirm.title")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t("overwriteConfirm.description")}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogClose onClick={onCancel} variant="outline">
            {t("overwriteConfirm.cancel")}
          </AlertDialogClose>
          <AlertDialogClose onClick={onConfirm}>
            {t("overwriteConfirm.confirm")}
          </AlertDialogClose>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
