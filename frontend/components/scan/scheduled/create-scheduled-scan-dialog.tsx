"use client"

import React from "react"
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Button } from "@/components/ui/button"
import { EdgePanelHeader } from "@/components/shared/edge-panel-header"
import { semanticIcons } from "@/components/icons"
import {
  scanWorkbenchDrawerContentClassName,
} from "@/lib/ui/overlay-styles"
import {
  ScheduledScanFooter,
  ScheduledScanOverwriteDialog,
  ScheduledScanPresetInfoStep,
  ScheduledScanScheduleStep,
  ScheduledScanScopeStep,
} from "@/components/scan/scheduled/scheduled-scan-dialog-sections"
import { useScheduledScanDialogState } from "@/components/scan/scheduled/scheduled-scan-dialog-state"
import { ScanAgentSelector } from "@/components/scan/agent-selector"
import { ScanInputSourceSelector } from "@/components/scan/scan-input-source-selector"
import {
  InitiateScanConfigStep,
  InitiateScanWorkflowSelection,
} from "@/components/scan/initiate-scan-dialog-sections"
import { useTranslations, useLocale } from "next-intl"
import type { Locale } from "@/i18n/config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

interface CreateScheduledScanSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
  presetOrganizationId?: number
  presetOrganizationName?: string
  presetTargetId?: number
  presetTargetName?: string
}

export function CreateScheduledScanSheet({
  open,
  onOpenChange,
  onSuccess,
  presetOrganizationId,
  presetOrganizationName,
  presetTargetId,
  presetTargetName,
}: CreateScheduledScanSheetProps) {
  const t = useTranslations("scan.scheduled")
  const tInitiate = useTranslations("scan.initiate")
  const tActions = useTranslations("common.actions")
  const locale = useLocale() as Locale

  const CRON_PRESETS = [
    { label: t("presets.everyHour"), value: "0 * * * *" },
    { label: t("presets.daily2am"), value: "0 2 * * *" },
    { label: t("presets.daily4am"), value: "0 4 * * *" },
    { label: t("presets.weekly"), value: "0 2 * * 1" },
    { label: t("presets.monthly"), value: "0 2 1 * *" },
  ]

  const hasPreset = !!(presetOrganizationId || presetTargetId)
  const totalSteps = 4

  const {
    isPending,
    configValidationRef,
    orgSearchInput,
    setOrgSearchInput,
    orgPageSize,
    setOrgPageSize,
    organizationTotalCount,
    organizationPaginationNavigation,
    onFirstOrgPage,
    onPreviousOrgPage,
    onNextOrgPage,
    targetSearchInput,
    setTargetSearchInput,
    targetPageSize,
    setTargetPageSize,
    targetTotalCount,
    targetPaginationNavigation,
    onFirstTargetPage,
    onPreviousTargetPage,
    onNextTargetPage,
    isOrgFetching,
    isTargetFetching,
    currentStep,
    goToPrevStep,
    name,
    setName,
    selectedScanWorkflowName,
    isLoadingWorkflows,
    isWorkflowsError,
    isWorkflowConfigLoading,
    selectionMode,
    setSelectionMode,
    selectedOrgId,
    selectedTargetId,
    selectedAgentID,
    inputSource,
    setSelectedOrgId,
    setSelectedTargetId,
    setSelectedAgentID,
    setInputSource,
    cronExpression,
    setCronExpression,
    configuration,
    isConfigEdited,
    isYamlValid,
    showOverwriteConfirm,
    targets,
    workflows,
    organizations,
    selectedScanWorkflows,
    engineCatalogDetails,
    isEngineCatalogLoading,
    isEngineCatalogError,
    handleConfigSync,
    handleManualConfigChange,
    handleWorkflowNamesChange,
    handleResetWorkflowConfig,
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
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetContent
        side="right"
        showCloseButton={false}
        className={scanWorkbenchDrawerContentClassName}
      >
        <SheetHeader className="shrink-0 border-b bg-card px-5 pt-4 text-left sm:px-6">
          <EdgePanelHeader
            variant="workbench"
            leading={<semanticIcons.concept.scheduledScan className="size-7" />}
            title={<SheetTitle className={textRole.sectionTitle}>{t("createTitle")}</SheetTitle>}
            description={<SheetDescription className={textRole.helperText}>{t("createDesc")}</SheetDescription>}
            actions={(
              <>
                <span className={cn("hidden sm:inline", textRole.helperText)}>
                  {t("stepIndicator", { current: currentStep, total: totalSteps })}
                </span>
                <SheetClose
                  render={(
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      aria-label={tActions("close")}
                      disabled={isPending}
                    />
                  )}
                >
                  <semanticIcons.action.cancel />
                </SheetClose>
              </>
            )}
          />
        </SheetHeader>

        <div className="flex-1 min-h-0 min-w-0 max-w-full overflow-hidden">
          {currentStep === 1 && (
            <div className="h-full min-w-0 max-w-full space-y-6 overflow-y-auto p-6">
              {hasPreset ? (
                <ScheduledScanPresetInfoStep
                  t={t}
                  name={name}
                  setName={setName}
                  presetTargetName={presetTargetName}
                  presetOrganizationName={presetOrganizationName}
                  presetTargetId={presetTargetId}
                />
              ) : (
                <ScheduledScanScopeStep
                  t={t}
                  name={name}
                  setName={setName}
                  selectionMode={selectionMode}
                  setSelectionMode={setSelectionMode}
                  orgSearchInput={orgSearchInput}
                  setOrgSearchInput={setOrgSearchInput}
                  orgPageSize={orgPageSize}
                  setOrgPageSize={setOrgPageSize}
                  organizationTotalCount={organizationTotalCount}
                  organizationPaginationNavigation={organizationPaginationNavigation}
                  onFirstOrgPage={onFirstOrgPage}
                  onPreviousOrgPage={onPreviousOrgPage}
                  onNextOrgPage={onNextOrgPage}
                  targetSearchInput={targetSearchInput}
                  setTargetSearchInput={setTargetSearchInput}
                  targetPageSize={targetPageSize}
                  setTargetPageSize={setTargetPageSize}
                  targetTotalCount={targetTotalCount}
                  targetPaginationNavigation={targetPaginationNavigation}
                  onFirstTargetPage={onFirstTargetPage}
                  onPreviousTargetPage={onPreviousTargetPage}
                  onNextTargetPage={onNextTargetPage}
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
            </div>
          )}

          {/* Step 2: Reuse the quick-scan workflow and Agent selection surfaces. */}
          {currentStep === 2 && (
            <div className="h-full min-w-0 max-w-full overflow-y-auto p-6 space-y-6">
              <InitiateScanWorkflowSelection
                t={tInitiate}
                workflows={workflows}
                selectedWorkflowNames={selectedScanWorkflowName ? [selectedScanWorkflowName] : []}
                isLoadingWorkflows={isLoadingWorkflows}
                isWorkflowsError={isWorkflowsError}
                isSubmitting={isPending || isWorkflowConfigLoading}
                onWorkflowNamesChange={(workflowNames) => void handleWorkflowNamesChange(workflowNames)}
              />
              <div className="space-y-5 border-t pt-6">
                <ScanInputSourceSelector
                  id="create-scheduled-scan-input-source"
                  value={inputSource}
                  onValueChange={setInputSource}
                  disabled={isPending}
                />
                <ScanAgentSelector value={selectedAgentID} onChange={setSelectedAgentID} disabled={isPending} />
              </div>
            </div>
          )}

          {/* Step 3: Reuse the quick-scan scan-options editor. */}
          {currentStep === 3 && (
            <div className="flex h-full min-w-0 max-w-full flex-col overflow-hidden p-6">
              <InitiateScanConfigStep
                t={tInitiate}
                configuration={configuration}
                selectedWorkflows={selectedScanWorkflows}
                isConfigEdited={isConfigEdited}
                isYamlValid={isYamlValid}
                hasConfig={configuration.trim().length > 0}
                selectMode="custom"
                selectedWorkflowNames={selectedScanWorkflowName ? [selectedScanWorkflowName] : []}
                locale={locale}
                engineCatalogDetails={engineCatalogDetails}
                isEngineCatalogLoading={isEngineCatalogLoading}
                isEngineCatalogError={isEngineCatalogError}
                isSubmitting={isPending || isWorkflowConfigLoading}
                onConfigSync={handleConfigSync}
                onConfigChange={handleManualConfigChange}
                onResetConfig={() => void handleResetWorkflowConfig()}
                onYamlValidationChange={handleYamlValidationChange}
                configValidationRef={configValidationRef}
              />
            </div>
          )}

          {/* Step 4: Keep schedule configuration separate from scan scope and options. */}
          {currentStep === 4 && (
            <div className="h-full min-w-0 max-w-full overflow-y-auto p-6">
              <ScheduledScanScheduleStep
                t={t}
                cronExpression={cronExpression}
                setCronExpression={setCronExpression}
                cronPresets={CRON_PRESETS}
                getCronDescription={getCronDescription}
                getNextExecutions={getNextExecutions}
                disabled={isPending}
              />
            </div>
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
      </SheetContent>
      
      {/* Overwrite confirmation dialog */}
      <ScheduledScanOverwriteDialog
        t={t}
        open={showOverwriteConfirm}
        onOpenChange={handleOverwriteDialogChange}
        onCancel={handleOverwriteCancel}
        onConfirm={handleOverwriteConfirm}
      />
    </Sheet>
  )
}

export const CreateScheduledScanDialog = CreateScheduledScanSheet
