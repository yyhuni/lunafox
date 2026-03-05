"use client"

import React from "react"
import { FileCode } from "@/components/icons"
import { useTranslations } from "next-intl"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { cn } from "@/lib/utils"
import type { WorkflowProfile } from "@/types/workflow.types"
import { useWorkflowCreateDialogState } from "@/components/scan/workflow/workflow-create-dialog-state"
import {
  WorkflowCreateConfigStep,
  WorkflowCreatePresetStep,
} from "@/components/scan/workflow/workflow-create-dialog-sections"

interface WorkflowCreateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSave?: (name: string, yamlContent: string) => Promise<void>
  /** Pre-selected preset (skip step 1) */
  preSelectedPreset?: WorkflowProfile
}

/**
 * Create new workflow template dialog - requires selecting a preset template
 */
export function WorkflowCreateDialog({
  open,
  onOpenChange,
  onSave,
  preSelectedPreset,
}: WorkflowCreateDialogProps) {
  const t = useTranslations("scan.workflow.create")
  const tWorkflow = useTranslations("scan.workflow")
  const tToast = useTranslations("toast")
  const tCommon = useTranslations("common.actions")
  const {
    step,
    presetWorkflows,
    selectedPreset,
    setSelectedPreset,
    workflowName,
    setWorkflowName,
    yamlContent,
    yamlError,
    isSubmitting,
    handleEditorChange,
    handleSave,
    handleNextStep,
    handleBackToStep1,
    handleClose,
  } = useWorkflowCreateDialogState({
    open,
    onOpenChange,
    onSave,
    preSelectedPreset,
    t,
    tToast,
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-4xl max-w-[calc(100%-2rem)] h-[85vh] flex flex-col p-0">
        <div className="flex flex-col h-full">
          <DialogHeader className="px-6 pt-6 pb-4 border-b">
            <DialogTitle className="flex items-center gap-2">
              <FileCode className="h-5 w-5" />
              {t("title")}
            </DialogTitle>
            <DialogDescription>
              {step === 1 ? t("selectPresetDesc") : t("editConfigDesc")}
            </DialogDescription>
            {/* Step indicator */}
            <div className="flex items-center gap-2 mt-3">
              <div className={cn(
                "flex items-center justify-center w-6 h-6 rounded-full text-xs font-medium",
                step === 1 ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"
              )}>
                1
              </div>
              <div className="w-8 h-px bg-border" />
              <div className={cn(
                "flex items-center justify-center w-6 h-6 rounded-full text-xs font-medium",
                step === 2 ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"
              )}>
                2
              </div>
            </div>
          </DialogHeader>

          {step === 1 ? (
            <WorkflowCreatePresetStep
              presetWorkflows={presetWorkflows}
              selectedPreset={selectedPreset}
              onSelectPreset={setSelectedPreset}
              onCancel={handleClose}
              onNext={handleNextStep}
              t={t}
              tWorkflow={tWorkflow}
              tCommon={tCommon}
            />
          ) : (
            <WorkflowCreateConfigStep
              workflowName={workflowName}
              onWorkflowNameChange={setWorkflowName}
              yamlContent={yamlContent}
              onYamlChange={handleEditorChange}
              yamlError={yamlError}
              isSubmitting={isSubmitting}
              preSelectedPreset={preSelectedPreset}
              onBack={handleBackToStep1}
              onCancel={handleClose}
              onSave={handleSave}
              t={t}
              tCommon={tCommon}
              tToast={tToast}
            />
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
