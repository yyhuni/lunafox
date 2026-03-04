"use client"

import { AlertCircle, ArrowLeft, ArrowRight, Check, CheckCircle2, Lock, Save, X } from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { CodeEditor } from "@/components/ui/code-editor"
import { DialogFooter } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ScrollArea } from "@/components/ui/scroll-area"
import { cn } from "@/lib/utils"
import { parseWorkflowCapabilities } from "@/lib/workflow-config"
import type { PresetWorkflow } from "@/types/workflow.types"
import type { WorkflowYamlError } from "@/components/scan/workflow/workflow-create-dialog-state"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface WorkflowCreatePresetStepProps {
  presetWorkflows: PresetWorkflow[]
  selectedPreset: PresetWorkflow | null
  onSelectPreset: (preset: PresetWorkflow) => void
  onCancel: () => void
  onNext: () => void
  t: TranslationFn
  tWorkflow: TranslationFn
  tCommon: TranslationFn
}

export function WorkflowCreatePresetStep({
  presetWorkflows,
  selectedPreset,
  onSelectPreset,
  onCancel,
  onNext,
  t,
  tWorkflow,
  tCommon,
}: WorkflowCreatePresetStepProps) {
  return (
    <>
      <ScrollArea className="flex-1 px-6 py-4">
        <div className="grid grid-cols-2 gap-3">
          {presetWorkflows.map((preset) => (
            <button type="button"
              key={preset.id}
              onClick={() => onSelectPreset(preset)}
              aria-label={preset.name}
              className={cn(
                "text-left p-4 rounded-lg border-2 transition-[background-color,border-color,color,box-shadow]",
                selectedPreset?.id === preset.id
                  ? "border-primary bg-primary/5"
                  : "border-border hover:border-primary/50 hover:bg-muted/50"
              )}
            >
              <div className="flex items-start gap-3">
                <div className={cn(
                  "flex h-8 w-8 items-center justify-center rounded-lg shrink-0",
                  selectedPreset?.id === preset.id ? "bg-primary/10" : "bg-muted"
                )}>
                  {selectedPreset?.id === preset.id ? (
                    <Check className="h-4 w-4 text-primary" />
                  ) : (
                    <Lock className="h-4 w-4 text-muted-foreground" />
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="font-medium text-sm">{preset.name}</div>
                  <p className="text-xs text-muted-foreground mt-1 line-clamp-2">
                    {preset.description}
                  </p>
                  <div className="flex flex-wrap gap-1 mt-2">
                    {(() => {
                      const features = parseWorkflowCapabilities(preset.configuration)
                      return (
                        <>
                          {features.slice(0, 3).map((feature) => (
                            <Badge key={feature} variant="secondary" className="text-[10px] px-1.5 py-0">
                              {tWorkflow(`features.${feature}`)}
                            </Badge>
                          ))}
                          {features.length > 3 && (
                            <Badge variant="secondary" className="text-[10px] px-1.5 py-0">
                              +{features.length - 3}
                            </Badge>
                          )}
                        </>
                      )
                    })()}
                  </div>
                </div>
              </div>
            </button>
          ))}
        </div>
      </ScrollArea>

      <DialogFooter className="px-6 py-4 border-t gap-2">
        <Button
          type="button"
          variant="outline"
          onClick={onCancel}
        >
          <X className="h-4 w-4" />
          {tCommon("cancel")}
        </Button>
        <Button
          type="button"
          onClick={onNext}
          disabled={!selectedPreset}
        >
          {t("nextStep")}
          <ArrowRight className="h-4 w-4" />
        </Button>
      </DialogFooter>
    </>
  )
}

interface WorkflowCreateConfigStepProps {
  workflowName: string
  onWorkflowNameChange: (value: string) => void
  yamlContent: string
  onYamlChange: (value: string) => void
  yamlError: WorkflowYamlError
  isSubmitting: boolean
  preSelectedPreset?: PresetWorkflow
  onBack: () => void
  onCancel: () => void
  onSave: () => void
  t: TranslationFn
  tCommon: TranslationFn
  tToast: TranslationFn
}

export function WorkflowCreateConfigStep({
  workflowName,
  onWorkflowNameChange,
  yamlContent,
  onYamlChange,
  yamlError,
  isSubmitting,
  preSelectedPreset,
  onBack,
  onCancel,
  onSave,
  t,
  tCommon,
  tToast,
}: WorkflowCreateConfigStepProps) {
  return (
    <>
      <div className="flex-1 overflow-hidden px-6 py-4">
        <div className="flex flex-col h-full gap-4">
          <div className="space-y-2">
            <Label htmlFor="workflow-name">
              {t("engineName")} <span className="text-destructive">*</span>
            </Label>
            <Input
              id="workflow-name"
              name="workflowName"
              autoComplete="off"
              value={workflowName}
              onChange={(event) => onWorkflowNameChange(event.target.value)}
              placeholder={t("engineNamePlaceholder")}
              disabled={isSubmitting}
              className="max-w-md"
            />
          </div>

          <div className="flex flex-col flex-1 min-h-0 gap-2">
            <div className="flex items-center justify-between">
              <Label>{t("yamlConfig")}</Label>
              <div className="flex items-center gap-2">
                {yamlContent.trim() && (
                  yamlError ? (
                    <div className="flex items-center gap-1 text-xs text-destructive">
                      <AlertCircle className="h-3.5 w-3.5" />
                      <span>{t("syntaxError")}</span>
                    </div>
                  ) : (
                    <div className="flex items-center gap-1 text-xs text-green-600 dark:text-green-400">
                      <CheckCircle2 className="h-3.5 w-3.5" />
                      <span>{t("syntaxValid")}</span>
                    </div>
                  )
                )}
              </div>
            </div>

            <CodeEditor
              value={yamlContent}
              onChange={onYamlChange}
              language="yaml"
              readOnly={isSubmitting}
              className={`flex-1 ${yamlError ? "border-destructive" : ""}`}
              showLineNumbers
              showFoldGutter
            />

            {yamlError && (
              <div className="flex items-start gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-md">
                <AlertCircle className="h-4 w-4 text-destructive mt-0.5 flex-shrink-0" />
                <div className="flex-1 text-xs">
                  <p className="font-semibold text-destructive mb-1">
                    {yamlError.line && yamlError.column
                      ? t("errorLocation", { line: yamlError.line, column: yamlError.column })
                      : tToast("yamlSyntaxError")}
                  </p>
                  <p className="text-muted-foreground">{yamlError.message}</p>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      <DialogFooter className="px-6 py-4 border-t">
        <div className="flex items-center justify-between w-full">
          {!preSelectedPreset ? (
            <Button
              type="button"
              variant="ghost"
              onClick={onBack}
              disabled={isSubmitting}
            >
              <ArrowLeft className="h-4 w-4" />
              {t("prevStep")}
            </Button>
          ) : (
            <div />
          )}
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={onCancel}
              disabled={isSubmitting}
            >
              <X className="h-4 w-4" />
              {tCommon("cancel")}
            </Button>
            <Button
              type="button"
              onClick={onSave}
              disabled={isSubmitting || !workflowName.trim() || !!yamlError}
            >
              {isSubmitting ? (
                <>
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
                  {t("creating")}
                </>
              ) : (
                <>
                  <Save className="h-4 w-4" />
                  {t("createEngine")}
                </>
              )}
            </Button>
          </div>
        </div>
      </DialogFooter>
    </>
  )
}
