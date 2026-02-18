"use client"

import React from "react"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  ScheduledScanBasicInfoStep,
  ScheduledScanConfigStep,
  ScheduledScanEngineStep,
  ScheduledScanFooter,
  ScheduledScanOverwriteDialog,
  ScheduledScanPresetInfoStep,
  ScheduledScanScheduleStep,
  ScheduledScanTargetSelectionStep,
} from "@/components/scan/scheduled/scheduled-scan-dialog-sections"
import { useScheduledScanDialogState } from "@/components/scan/scheduled/scheduled-scan-dialog-state"
import { useTranslations, useLocale } from "next-intl"

interface CreateScheduledScanDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
  presetOrganizationId?: number
  presetOrganizationName?: string
  presetTargetId?: number
  presetTargetName?: string
}

export function CreateScheduledScanDialog({
  open,
  onOpenChange,
  onSuccess,
  presetOrganizationId,
  presetOrganizationName,
  presetTargetId,
  presetTargetName,
}: CreateScheduledScanDialogProps) {
  const t = useTranslations("scan.scheduled")
  const locale = useLocale()

  const CRON_PRESETS = [
    { label: t("presets.everyHour"), value: "0 * * * *" },
    { label: t("presets.daily2am"), value: "0 2 * * *" },
    { label: t("presets.daily4am"), value: "0 4 * * *" },
    { label: t("presets.weekly"), value: "0 2 * * 1" },
    { label: t("presets.monthly"), value: "0 2 1 * *" },
  ]

  const hasPreset = !!(presetOrganizationId || presetTargetId)
  const totalSteps = hasPreset ? 4 : 5

  const {
    isPending,
    orgSearchInput,
    setOrgSearchInput,
    targetSearchInput,
    setTargetSearchInput,
    handleOrgSearch,
    handleTargetSearch,
    isOrgFetching,
    isTargetFetching,
    currentStep,
    goToPrevStep,
    name,
    setName,
    engineIds,
    selectedPresetId,
    setSelectedPresetId,
    selectionMode,
    setSelectionMode,
    selectedOrgId,
    selectedTargetId,
    setSelectedOrgId,
    setSelectedTargetId,
    cronExpression,
    setCronExpression,
    configuration,
    isConfigEdited,
    showOverwriteConfirm,
    targets,
    engines,
    organizations,
    selectedEngines,
    handlePresetConfigChange,
    handleManualConfigChange,
    handleEngineIdsChange,
    handleOverwriteConfirm,
    handleOverwriteCancel,
    handleYamlValidationChange,
    handleOpenChange,
    handleOrgSelect,
    handleTargetSelect,
    handleNext,
    handleSubmit,
    getCronDescription,
    getNextExecutions,
    setShowOverwriteConfirm,
  } = useScheduledScanDialogState({
    open,
    onOpenChange,
    onSuccess,
    presetOrganizationId,
    presetOrganizationName,
    presetTargetId,
    presetTargetName,
    hasPreset,
    totalSteps,
    locale,
    t,
  })

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
      <DialogContent className="max-w-[900px] p-0 gap-0">
        <DialogHeader className="px-6 pt-6 pb-4">
          <div className="flex items-center justify-between">
            <div>
              <DialogTitle>{t("createTitle")}</DialogTitle>
              <DialogDescription className="mt-1">{t("createDesc")}</DialogDescription>
            </div>
            {/* Step indicator */}
            <div className="text-sm text-muted-foreground mr-8">
              {t("stepIndicator", { current: currentStep, total: totalSteps })}
            </div>
          </div>
        </DialogHeader>

        <div className="border-t h-[480px] overflow-hidden">
          {/* Step 1: Basic Info + Scan Mode (full mode only) */}
          {currentStep === 1 && !hasPreset && (
            <ScheduledScanBasicInfoStep
              t={t}
              name={name}
              setName={setName}
              selectionMode={selectionMode}
              setSelectionMode={setSelectionMode}
              setSelectedOrgId={setSelectedOrgId}
              setSelectedTargetId={setSelectedTargetId}
            />
          )}

          {/* Step 1: Basic Info (preset mode - name only, target is locked) */}
          {currentStep === 1 && hasPreset && (
            <ScheduledScanPresetInfoStep
              t={t}
              name={name}
              setName={setName}
              presetTargetName={presetTargetName}
              presetOrganizationName={presetOrganizationName}
              presetTargetId={presetTargetId}
            />
          )}

          {/* Step 2: Select Target (Organization or Target) */}
          {currentStep === 2 && !hasPreset && (
            <ScheduledScanTargetSelectionStep
              t={t}
              selectionMode={selectionMode}
              orgSearchInput={orgSearchInput}
              setOrgSearchInput={setOrgSearchInput}
              targetSearchInput={targetSearchInput}
              setTargetSearchInput={setTargetSearchInput}
              handleOrgSearch={handleOrgSearch}
              handleTargetSearch={handleTargetSearch}
              isOrgFetching={isOrgFetching}
              isTargetFetching={isTargetFetching}
              organizations={organizations}
              targets={targets}
              selectedOrgId={selectedOrgId}
              selectedTargetId={selectedTargetId}
              setSelectedOrgId={setSelectedOrgId}
              setSelectedTargetId={setSelectedTargetId}
              onSelectOrg={handleOrgSelect}
              onSelectTarget={handleTargetSelect}
            />
          )}

          {/* Step 3 (full) / Step 2 (preset): Select Engine */}
          {((currentStep === 3 && !hasPreset) || (currentStep === 2 && hasPreset)) && (
            <ScheduledScanEngineStep
              engines={engines}
              engineIds={engineIds}
              selectedPresetId={selectedPresetId}
              setSelectedPresetId={setSelectedPresetId}
              onEngineIdsChange={handleEngineIdsChange}
              onConfigChange={handlePresetConfigChange}
              disabled={isPending}
            />
          )}

          {/* Step 4 (full) / Step 3 (preset): Edit Configuration */}
          {((currentStep === 4 && !hasPreset) || (currentStep === 3 && hasPreset)) && (
            <ScheduledScanConfigStep
              configuration={configuration}
              onConfigChange={handleManualConfigChange}
              onValidationChange={handleYamlValidationChange}
              selectedEngines={selectedEngines}
              isConfigEdited={isConfigEdited}
              disabled={isPending}
            />
          )}

          {/* Step 5 (full) / Step 4 (preset): Schedule Settings */}
          {((currentStep === 5 && !hasPreset) || (currentStep === 4 && hasPreset)) && (
            <ScheduledScanScheduleStep
              t={t}
              cronExpression={cronExpression}
              setCronExpression={setCronExpression}
              cronPresets={CRON_PRESETS}
              getCronDescription={getCronDescription}
              getNextExecutions={getNextExecutions}
            />
          )}
        </div>

        <ScheduledScanFooter
          t={t}
          currentStep={currentStep}
          totalSteps={totalSteps}
          isPending={isPending}
          onPrev={goToPrevStep}
          onNext={handleNext}
          onSubmit={handleSubmit}
        />
      </DialogContent>
      
      {/* Overwrite confirmation dialog */}
      <ScheduledScanOverwriteDialog
        t={t}
        open={showOverwriteConfirm}
        onOpenChange={handleOverwriteDialogChange}
        onCancel={handleOverwriteCancel}
        onConfirm={handleOverwriteConfirm}
      />
    </Dialog>
  )
}
