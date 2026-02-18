"use client"

import React from "react"
import { useTranslations } from "next-intl"
import { Dialog, DialogContent } from "@/components/ui/dialog"
import type { ScanEngine } from "@/types/engine.types"
import { useEngineEditDialogState } from "@/components/scan/engine/engine-edit-dialog-state"
import {
  EngineEditEditor,
  EngineEditFooter,
  EngineEditHeader,
} from "@/components/scan/engine/engine-edit-dialog-sections"

interface EngineEditDialogProps {
  engine: ScanEngine | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onSave?: (engineId: number, yamlContent: string) => Promise<void>
}

/**
 * Engine configuration edit dialog
 * Uses Monaco Editor to provide VSCode-level editing experience
 */
export function EngineEditDialog({
  engine,
  open,
  onOpenChange,
  onSave,
}: EngineEditDialogProps) {
  const t = useTranslations("scan.engine.edit")
  const tToast = useTranslations("toast")
  const tCommon = useTranslations("common.actions")
  const {
    yamlContent,
    isSubmitting,
    hasChanges,
    yamlError,
    handleEditorChange,
    handleSave,
    handleClose,
  } = useEngineEditDialogState({
    engine,
    open,
    onOpenChange,
    onSave,
    t,
    tToast,
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-6xl max-w-[calc(100%-2rem)] h-[90vh] flex flex-col p-0">
        <div className="flex flex-col h-full">
          <EngineEditHeader engineName={engine?.name} t={t} />
          <EngineEditEditor
            t={t}
            tToast={tToast}
            yamlContent={yamlContent}
            yamlError={yamlError}
            isSubmitting={isSubmitting}
            onChange={handleEditorChange}
          />
          <EngineEditFooter
            t={t}
            tCommon={tCommon}
            isSubmitting={isSubmitting}
            hasChanges={hasChanges}
            yamlError={yamlError}
            onCancel={handleClose}
            onSave={handleSave}
          />
        </div>
      </DialogContent>
    </Dialog>
  )
}
