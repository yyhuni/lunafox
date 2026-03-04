"use client"

import React from "react"
import { Play } from "@/components/icons"
import { useTranslations } from "next-intl"

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { useInitiateScanDialogState } from "@/components/scan/initiate-scan-dialog-state"
import {
  InitiateScanConfigStep,
  InitiateScanWorkflowSelection,
  InitiateScanFooter,
  InitiateScanOverwriteDialog,
  InitiateScanStepHeader,
} from "@/components/scan/initiate-scan-dialog-sections"
import type { Organization } from "@/types/organization.types"

interface InitiateScanDialogProps {
  organization?: Organization | null
  organizationId?: number
  targetId?: number
  targetName?: string
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

export function InitiateScanDialog({
  organization,
  organizationId,
  targetId,
  targetName,
  open,
  onOpenChange,
  onSuccess,
}: InitiateScanDialogProps) {
  const t = useTranslations("scan.initiate")
  const tToast = useTranslations("toast")
  const {
    workflows,
    isLoadingWorkflows,
    isWorkflowsError,
    presetWorkflows,
    isLoadingPresets,
    isPresetsError,
    selectedWorkflowIds,
    selectedPresetId,
    selectMode,
    isSubmitting,
    currentStep,
    configuration,
    isConfigEdited,
    isYamlValid,
    showOverwriteConfirm,
    selectedWorkflows,
    selectedPreset,
    hasConfig,
    canProceedToReview,
    canStart,
    setCurrentStep,
    handleManualConfigChange,
    handleWorkflowIdsChange,
    handlePresetSelect,
    handleOverwriteConfirm,
    handleOverwriteCancel,
    handleYamlValidationChange,
    handleSelectModeChange,
    handleInitiate,
    handleOpenChange,
    setShowOverwriteConfirm,
  } = useInitiateScanDialogState({
    organizationId,
    targetId,
    onOpenChange,
    onSuccess,
    tToast,
  })

  const steps = [
    { id: 1, title: t("steps.selectEngine") },
    { id: 2, title: t("steps.editConfig") },
  ]

  const handleOverwriteDialogChange = React.useCallback(
    (nextOpen: boolean) => {
      if (!nextOpen) {
        handleOverwriteCancel()
      } else {
        setShowOverwriteConfirm(true)
      }
    },
    [handleOverwriteCancel, setShowOverwriteConfirm]
  )

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-[90vw] sm:max-w-[900px] p-0 gap-0 flex flex-col max-h-[90vh]">
        <DialogHeader className="px-6 pt-6 pb-4 shrink-0">
          <DialogTitle className="flex items-center gap-2">
            <Play className="h-5 w-5" />
            {t("title")}
          </DialogTitle>
          <DialogDescription className="mt-1">
            {targetName ? (
              <>{t("targetDesc")} <span className="font-medium text-foreground">{targetName}</span></>
            ) : (
              <>{t("orgDesc")} <span className="font-medium text-foreground">{organization?.name}</span></>
            )}
          </DialogDescription>
        </DialogHeader>

        {/* Scrollable content area */}
        <div className="flex-1 overflow-y-auto border-t">
          <>
            <InitiateScanStepHeader steps={steps} currentStep={currentStep} />

            {currentStep === 1 && (
              <InitiateScanWorkflowSelection
                t={t}
                selectMode={selectMode}
                selectedPresetId={selectedPresetId}
                selectedPreset={selectedPreset}
                presetWorkflows={presetWorkflows}
                isLoadingPresets={isLoadingPresets}
                isPresetsError={isPresetsError}
                selectedWorkflowIds={selectedWorkflowIds}
                workflows={workflows}
                isLoadingWorkflows={isLoadingWorkflows}
                isWorkflowsError={isWorkflowsError}
                isSubmitting={isSubmitting}
                onSelectModeChange={handleSelectModeChange}
                onPresetSelect={handlePresetSelect}
                onWorkflowIdsChange={handleWorkflowIdsChange}
              />
            )}

            {currentStep === 2 && (
              <InitiateScanConfigStep
                t={t}
                configuration={configuration}
                selectedWorkflows={selectedWorkflows}
                isConfigEdited={isConfigEdited}
                isYamlValid={isYamlValid}
                hasConfig={hasConfig}
                selectMode={selectMode}
                selectedPreset={selectedPreset}
                selectedWorkflowIds={selectedWorkflowIds}
                isSubmitting={isSubmitting}
                onConfigChange={handleManualConfigChange}
                onYamlValidationChange={handleYamlValidationChange}
              />
            )}
          </>
        </div>

        {/* Sticky footer */}
        <InitiateScanFooter
          t={t}
          currentStep={currentStep}
          selectMode={selectMode}
          selectedWorkflowIds={selectedWorkflowIds}
          selectedPreset={selectedPreset}
          canProceedToReview={canProceedToReview}
          canStart={canStart}
          isSubmitting={isSubmitting}
          onCancel={() => onOpenChange(false)}
          onBack={() => setCurrentStep((prev) => Math.max(1, prev - 1))}
          onNext={() => setCurrentStep(2)}
          onStart={handleInitiate}
        />
      </DialogContent>
      
      {/* Overwrite confirmation dialog */}
      <InitiateScanOverwriteDialog
        t={t}
        open={showOverwriteConfirm}
        onOpenChange={handleOverwriteDialogChange}
        onCancel={handleOverwriteCancel}
        onConfirm={handleOverwriteConfirm}
      />
    </Dialog>
  )
}
