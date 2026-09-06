"use client"

import React from "react"
import { useTranslations } from "next-intl"
import { Dialog, DialogContent } from "@/components/ui/dialog"
import { editorDialogPanelClassName } from "@/lib/ui/overlay-styles"
import { cn } from "@/lib/utils"
import type { ScanWorkflow } from "@/types/scan-workflow.types"
import { useWorkflowEditDialogState } from "@/components/scan/workflow/workflow-edit-dialog-state"
import {
  WorkflowEditEditor,
  WorkflowEditFooter,
  WorkflowEditHeader,
} from "@/components/scan/workflow/workflow-edit-dialog-sections"

interface WorkflowEditDialogProps {
  workflow: ScanWorkflow | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onSave?: (workflowName: string, yamlContent: string) => Promise<void>
}

/**
 * Workflow template configuration edit dialog
 * Uses Monaco Editor to provide VSCode-level editing experience
 */
export function WorkflowEditDialog({
  workflow,
  open,
  onOpenChange,
  onSave,
}: WorkflowEditDialogProps) {
  const t = useTranslations("scan.workflow.edit")
  const tToast = useTranslations("toast")
  const {
    yamlContent,
    isSubmitting,
    hasChanges,
    yamlError,
    handleEditorChange,
    handleSave,
  } = useWorkflowEditDialogState({
    workflow,
    open,
    onOpenChange,
    onSave,
    t,
    tToast,
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={cn(editorDialogPanelClassName, "sm:max-w-6xl")}>
        <div className="flex flex-col h-full">
          <WorkflowEditHeader workflowName={workflow?.name} t={t} />
          <WorkflowEditEditor
            t={t}
            tToast={tToast}
            yamlContent={yamlContent}
            yamlError={yamlError}
            isSubmitting={isSubmitting}
            onChange={handleEditorChange}
          />
          <WorkflowEditFooter
            t={t}
            isSubmitting={isSubmitting}
            hasChanges={hasChanges}
            yamlError={yamlError}
            onSave={handleSave}
          />
        </div>
      </DialogContent>
    </Dialog>
  )
}
