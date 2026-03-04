import React from "react"
import { AlertTriangle, CheckCircle2, ChevronRight, Play, Settings } from "@/components/icons"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { DialogFooter } from "@/components/ui/dialog"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { LoadingSpinner } from "@/components/loading-spinner"
import { ScanConfigEditor } from "./scan-config-editor"
import { cn } from "@/lib/utils"
import type { InitiateScanSelectMode } from "@/lib/initiate-scan-helpers"
import type { PresetWorkflow, ScanWorkflow } from "@/types/workflow.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

type StepDescriptor = {
  id: number
  title: string
}

interface InitiateScanStepHeaderProps {
  steps: StepDescriptor[]
  currentStep: number
}

export function InitiateScanStepHeader({ steps, currentStep }: InitiateScanStepHeaderProps) {
  return (
    <div className="px-6 pt-5 pb-2">
      <div className="flex flex-wrap items-center gap-3 text-sm">
        {steps.map((step, index) => (
          <React.Fragment key={step.id}>
            <div
              className={cn(
                "flex items-center gap-2",
                currentStep === step.id ? "text-foreground" : "text-muted-foreground"
              )}
            >
              <span
                className={cn(
                  "inline-flex h-7 w-7 items-center justify-center rounded-full border text-xs font-medium",
                  currentStep === step.id
                    ? "border-primary/40 bg-primary/10 text-primary"
                    : "border-muted-foreground/30 text-muted-foreground"
                )}
              >
                {step.id}
              </span>
              <span className={currentStep === step.id ? "font-medium" : undefined}>
                {step.title}
              </span>
            </div>
            {index < steps.length - 1 && (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            )}
          </React.Fragment>
        ))}
      </div>
    </div>
  )
}

interface InitiateScanWorkflowSelectionProps {
  t: TranslationFn
  selectMode: InitiateScanSelectMode
  selectedPresetId: string | null
  selectedPreset: PresetWorkflow | null
  presetWorkflows?: PresetWorkflow[]
  isLoadingPresets: boolean
  isPresetsError: boolean
  selectedWorkflowIds: number[]
  workflows?: ScanWorkflow[]
  isLoadingWorkflows: boolean
  isWorkflowsError: boolean
  isSubmitting: boolean
  onSelectModeChange: (value: string) => void
  onPresetSelect: (presetId: string, presetConfig: string) => void
  onWorkflowIdsChange: (workflowIds: number[]) => void
}

export function InitiateScanWorkflowSelection({
  t,
  selectMode,
  selectedPresetId,
  selectedPreset,
  presetWorkflows,
  isLoadingPresets,
  isPresetsError,
  selectedWorkflowIds,
  workflows,
  isLoadingWorkflows,
  isWorkflowsError,
  isSubmitting,
  onSelectModeChange,
  onPresetSelect,
  onWorkflowIdsChange,
}: InitiateScanWorkflowSelectionProps) {
  return (
    <div className="p-6 space-y-6">
      <Tabs value={selectMode} onValueChange={onSelectModeChange}>
        <TabsList>
          <TabsTrigger value="preset">{t("mode.preset")}</TabsTrigger>
          <TabsTrigger value="custom">{t("mode.custom")}</TabsTrigger>
        </TabsList>
        <TabsContent value="preset" className="mt-4 space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">{t("presets.title")}</p>
              <p className="text-xs text-muted-foreground">{t("presets.selectHint")}</p>
            </div>
            {selectedPreset && (
              <Badge variant="secondary" className="text-xs">
                {selectedPreset.name}
              </Badge>
            )}
          </div>
          {isLoadingPresets ? (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <LoadingSpinner />
              {t("presets.loading")}
            </div>
          ) : isPresetsError ? (
            <div className="text-sm text-destructive">{t("loadFailed")}</div>
          ) : presetWorkflows && presetWorkflows.length > 0 ? (
            <div className="grid gap-2 sm:grid-cols-2">
              {presetWorkflows.map((preset) => {
                const isSelected = preset.id === selectedPresetId
                return (
                  <button
                    key={preset.id}
                    type="button"
                    onClick={() => onPresetSelect(preset.id, preset.configuration || "")}
                    disabled={isSubmitting}
                    className={cn(
                      "flex flex-col items-start gap-2 rounded-lg border px-3 py-2 text-left transition-[background-color,border-color,color,box-shadow]",
                      isSelected
                        ? "border-primary/50 bg-primary/5"
                        : "border-border hover:border-primary/40 hover:bg-muted/30",
                      isSubmitting && "opacity-60 cursor-not-allowed"
                    )}
                  >
                    <span className="text-sm font-medium">{preset.name}</span>
                    {preset.description && (
                      <span className="text-xs text-muted-foreground line-clamp-2">
                        {preset.description}
                      </span>
                    )}
                  </button>
                )
              })}
            </div>
          ) : (
            <div className="text-sm text-muted-foreground">{t("presets.empty")}</div>
          )}
        </TabsContent>
        <TabsContent value="custom" className="mt-4 space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">{t("selectEngineTitle")}</p>
              <p className="text-xs text-muted-foreground">{t("selectWorkflowHint")}</p>
            </div>
            {selectedWorkflowIds.length > 0 && (
              <Badge variant="secondary" className="text-xs">
                {t("selectedCount", { count: selectedWorkflowIds.length })}
              </Badge>
            )}
          </div>
          {isLoadingWorkflows ? (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <LoadingSpinner />
              {t("loading")}
            </div>
          ) : isWorkflowsError ? (
            <div className="text-sm text-destructive">{t("loadFailed")}</div>
          ) : workflows && workflows.length > 0 ? (
            <div className="grid gap-2 sm:grid-cols-2">
              {workflows.map((workflow) => {
                const isSelected = selectedWorkflowIds.includes(workflow.id)
                return (
                  <label
                    key={workflow.id}
                    htmlFor={`initiate-workflow-${workflow.id}`}
                    className={cn(
                      "flex items-center gap-3 rounded-lg border px-3 py-2 cursor-pointer transition-[background-color,border-color,color,box-shadow]",
                      isSelected
                        ? "border-primary/50 bg-primary/5"
                        : "border-border hover:border-primary/40 hover:bg-muted/30"
                    )}
                  >
                    <Checkbox
                      id={`initiate-workflow-${workflow.id}`}
                      checked={isSelected}
                      onCheckedChange={(checked) => {
                        const nextIds = checked ? [workflow.id] : []
                        onWorkflowIdsChange(nextIds)
                      }}
                      disabled={isSubmitting}
                      className="h-4 w-4"
                    />
                    <div className="min-w-0">
                      <p className="text-sm font-medium truncate">{workflow.name}</p>
                      <p className="text-xs text-muted-foreground truncate">
                        {workflow.configuration ? t("configTitle") : t("noConfig")}
                      </p>
                    </div>
                  </label>
                )
              })}
            </div>
          ) : (
            <div className="text-sm text-muted-foreground">{t("noWorkflows")}</div>
          )}
        </TabsContent>
      </Tabs>
    </div>
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
  selectedPreset: PresetWorkflow | null
  selectedWorkflowIds: number[]
  isSubmitting: boolean
  onConfigChange: (value: string) => void
  onYamlValidationChange: (isValid: boolean) => void
}

export function InitiateScanConfigStep({
  t,
  configuration,
  selectedWorkflows,
  isConfigEdited,
  isYamlValid,
  hasConfig,
  selectMode,
  selectedPreset,
  selectedWorkflowIds,
  isSubmitting,
  onConfigChange,
  onYamlValidationChange,
}: InitiateScanConfigStepProps) {
  return (
    <div className="grid gap-6 p-6 lg:grid-cols-[1.1fr_0.9fr]">
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2 text-sm">
            <Settings className="h-4 w-4 text-muted-foreground" />
            <span className="font-medium">{t("configTitle")}</span>
          </div>
          {isConfigEdited && (
            <Badge variant="outline" className="text-xs">
              {t("configEdited")}
            </Badge>
          )}
        </div>
        <div className="border rounded-lg overflow-hidden">
          <ScanConfigEditor
            configuration={configuration}
            onChange={onConfigChange}
            onValidationChange={onYamlValidationChange}
            selectedWorkflows={selectedWorkflows}
            isConfigEdited={isConfigEdited}
            disabled={isSubmitting}
            className="h-[420px]"
          />
        </div>
      </div>

      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium">{t("validation.title")}</span>
          <Badge variant={isYamlValid && hasConfig ? "secondary" : "outline"} className="text-xs">
            {isYamlValid && hasConfig ? t("validation.yamlOk") : t("validation.yamlError")}
          </Badge>
        </div>
        <div className="rounded-lg border bg-muted/20 p-4 space-y-3 text-sm">
          <div className="flex items-start gap-2">
            {isYamlValid && hasConfig ? (
              <CheckCircle2 className="h-4 w-4 text-emerald-500 mt-0.5" />
            ) : (
              <AlertTriangle className="h-4 w-4 text-amber-500 mt-0.5" />
            )}
            <span>{isYamlValid && hasConfig ? t("validation.yamlOk") : t("validation.yamlError")}</span>
          </div>
          {selectMode === "preset" ? (
            <div className="flex items-start gap-2">
              {selectedPreset ? (
                <CheckCircle2 className="h-4 w-4 text-emerald-500 mt-0.5" />
              ) : (
                <AlertTriangle className="h-4 w-4 text-amber-500 mt-0.5" />
              )}
              <span>
                {selectedPreset
                  ? t("validation.presetOk", { name: selectedPreset.name })
                  : t("validation.presetMissing")}
              </span>
            </div>
          ) : (
            <div className="flex items-start gap-2">
              {selectedWorkflowIds.length > 0 ? (
                <CheckCircle2 className="h-4 w-4 text-emerald-500 mt-0.5" />
              ) : (
                <AlertTriangle className="h-4 w-4 text-amber-500 mt-0.5" />
              )}
              <span>
                {selectedWorkflowIds.length > 0
                  ? t("validation.enginesOk", { count: selectedWorkflowIds.length })
                  : t("validation.enginesMissing")}
              </span>
            </div>
          )}
          <div className="flex items-start gap-2">
            {hasConfig ? (
              <CheckCircle2 className="h-4 w-4 text-emerald-500 mt-0.5" />
            ) : (
              <AlertTriangle className="h-4 w-4 text-amber-500 mt-0.5" />
            )}
            <span>
              {hasConfig
                ? (isConfigEdited
                  ? t("validation.configEdited")
                  : selectMode === "preset"
                    ? t("validation.configFromPreset")
                    : t("validation.configFromEngine"))
                : t("validation.configMissing")}
            </span>
          </div>
        </div>
      </div>
    </div>
  )
}

interface InitiateScanFooterProps {
  t: TranslationFn
  currentStep: number
  selectMode: InitiateScanSelectMode
  selectedWorkflowIds: number[]
  selectedPreset: PresetWorkflow | null
  canProceedToReview: boolean
  canStart: boolean
  isSubmitting: boolean
  onCancel: () => void
  onBack: () => void
  onNext: () => void
  onStart: () => void
}

export function InitiateScanFooter({
  t,
  currentStep,
  selectMode,
  selectedWorkflowIds,
  selectedPreset,
  canProceedToReview,
  canStart,
  isSubmitting,
  onCancel,
  onBack,
  onNext,
  onStart,
}: InitiateScanFooterProps) {
  return (
    <DialogFooter className="px-6 py-4 border-t shrink-0 bg-background">
      <div className="flex items-center justify-between w-full">
        <div className="text-sm text-muted-foreground">
          {currentStep === 1 && selectMode === "custom" && selectedWorkflowIds.length > 0 && (
            <span className="text-primary">{t("selectedCount", { count: selectedWorkflowIds.length })}</span>
          )}
          {currentStep === 1 && selectMode === "preset" && selectedPreset && (
            <span className="text-primary">{selectedPreset.name}</span>
          )}
          {currentStep === 2 && (
            <span className={canStart ? "text-primary" : undefined}>
              {canStart ? t("validation.yamlOk") : t("validation.yamlError")}
            </span>
          )}
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={onCancel} disabled={isSubmitting}>
            {t("cancel")}
          </Button>
          {currentStep > 1 && (
            <Button variant="outline" onClick={onBack} disabled={isSubmitting}>
              {t("back")}
            </Button>
          )}
          {currentStep === 1 ? (
            <Button onClick={onNext} disabled={!canProceedToReview || isSubmitting}>
              {t("next")}
            </Button>
          ) : (
            <Button onClick={onStart} disabled={!canStart || isSubmitting}>
              {isSubmitting ? (
                <>
                  <LoadingSpinner />
                  {t("initiating")}
                </>
              ) : (
                <>
                  <Play className="h-4 w-4 mr-2" />
                  {t("startScan")}
                </>
              )}
            </Button>
          )}
        </div>
      </div>
    </DialogFooter>
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
          <AlertDialogCancel onClick={onCancel}>
            {t("overwriteConfirm.cancel")}
          </AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm}>
            {t("overwriteConfirm.confirm")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
