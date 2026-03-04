"use client"

import React from "react"
import { Dialog, DialogContent } from "@/components/ui/dialog"
import { useEditScheduledScanDialogState } from "@/components/scan/scheduled/edit-scheduled-scan-dialog-state"
import {
  EditScheduledScanDialogHeader,
  EditScheduledScanNameField,
  EditScheduledScanWorkflowSection,
  EditScheduledScanTargetSection,
  EditScheduledScanCronSection,
  EditScheduledScanDialogFooter,
} from "@/components/scan/scheduled/edit-scheduled-scan-dialog-sections"
import { useTranslations } from "next-intl"
import type { ScheduledScan } from "@/types/scheduled-scan.types"

interface EditScheduledScanDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  scheduledScan: ScheduledScan | null
  onSuccess?: () => void
}

export function EditScheduledScanDialog({
  open,
  onOpenChange,
  scheduledScan,
  onSuccess,
}: EditScheduledScanDialogProps) {
  const t = useTranslations("scan.scheduled")
  const {
    cronPresets,
    name,
    setName,
    workflowIds,
    selectedTargetId,
    cronExpression,
    setCronExpression,
    targets,
    workflows,
    isPending,
    handleWorkflowToggle,
    handleTargetSelect,
    handleSubmit,
  } = useEditScheduledScanDialogState({
    open,
    scheduledScan,
    onOpenChange,
    onSuccess,
    t,
  })

  if (!scheduledScan) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <EditScheduledScanDialogHeader t={t} />

        <div className="grid gap-6 py-4">
          <EditScheduledScanNameField
            t={t}
            name={name}
            onNameChange={setName}
          />

          <EditScheduledScanWorkflowSection
            t={t}
            workflows={workflows}
            workflowIds={workflowIds}
            onToggle={handleWorkflowToggle}
          />

          <EditScheduledScanTargetSection
            t={t}
            scheduledScan={scheduledScan}
            targets={targets}
            selectedTargetId={selectedTargetId}
            onSelectTarget={handleTargetSelect}
          />

          <EditScheduledScanCronSection
            t={t}
            cronExpression={cronExpression}
            onCronChange={setCronExpression}
            cronPresets={cronPresets}
            onPresetSelect={setCronExpression}
          />
        </div>

        <EditScheduledScanDialogFooter
          t={t}
          isPending={isPending}
          onCancel={() => onOpenChange(false)}
          onSubmit={handleSubmit}
        />
      </DialogContent>
    </Dialog>
  )
}
