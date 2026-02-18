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
import type { PresetEngine } from "@/types/engine.types"
import { useEngineCreateDialogState } from "@/components/scan/engine/engine-create-dialog-state"
import {
  EngineCreateConfigStep,
  EngineCreatePresetStep,
} from "@/components/scan/engine/engine-create-dialog-sections"

interface EngineCreateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSave?: (name: string, yamlContent: string) => Promise<void>
  /** Pre-selected preset (skip step 1) */
  preSelectedPreset?: PresetEngine
}

/**
 * Create new engine dialog - requires selecting a preset template
 */
export function EngineCreateDialog({
  open,
  onOpenChange,
  onSave,
  preSelectedPreset,
}: EngineCreateDialogProps) {
  const t = useTranslations("scan.engine.create")
  const tEngine = useTranslations("scan.engine")
  const tToast = useTranslations("toast")
  const tCommon = useTranslations("common.actions")
  const {
    step,
    presetEngines,
    selectedPreset,
    setSelectedPreset,
    engineName,
    setEngineName,
    yamlContent,
    yamlError,
    isSubmitting,
    handleEditorChange,
    handleSave,
    handleNextStep,
    handleBackToStep1,
    handleClose,
  } = useEngineCreateDialogState({
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
            <EngineCreatePresetStep
              presetEngines={presetEngines}
              selectedPreset={selectedPreset}
              onSelectPreset={setSelectedPreset}
              onCancel={handleClose}
              onNext={handleNextStep}
              t={t}
              tEngine={tEngine}
              tCommon={tCommon}
            />
          ) : (
            <EngineCreateConfigStep
              engineName={engineName}
              onEngineNameChange={setEngineName}
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
