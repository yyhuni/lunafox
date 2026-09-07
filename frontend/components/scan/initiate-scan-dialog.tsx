"use client"

import React from "react"
import { Play, semanticIcons } from "@/components/icons"
import { useLocale, useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer"
import { useInitiateScanDialogState } from "@/components/scan/initiate-scan-dialog-state"
import {
  InitiateScanConfigStep,
  InitiateScanExecutionOptions,
  InitiateScanWorkflowSelection,
  InitiateScanFooter,
  InitiateScanOverwriteDialog,
  InitiateScanStepHeader,
  countConfiguredWorkflowEngines,
  countWorkflowConfiguredEngines,
  getScanStepProgress,
} from "@/components/scan/initiate-scan-dialog-sections"
import { EdgePanelHeader } from "@/components/shared/edge-panel-header"
import {
  scanWorkbenchDrawerContentClassName,
} from "@/lib/ui/overlay-styles"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { Locale } from "@/i18n/config"
import type { Organization } from "@/types/organization.types"

interface InitiateScanDrawerProps {
  organization?: Organization | null
  organizationId?: number
  organizationIds?: number[]
  targetId?: number
  targetIds?: number[]
  targetName?: string
  bulkScopeLabel?: string
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

export function InitiateScanDrawer({
  organization,
  organizationId,
  organizationIds,
  targetId,
  targetIds,
  targetName,
  bulkScopeLabel,
  open,
  onOpenChange,
  onSuccess,
}: InitiateScanDrawerProps) {
  const t = useTranslations("scan.initiate")
  const tToast = useTranslations("toast")
  const tActions = useTranslations("common.actions")
  const locale = useLocale() as Locale
  const {
    workflows,
    isLoadingWorkflows,
    isWorkflowsError,
    selectedWorkflowNames,
    selectedAgentID,
    inputSource,
    selectMode,
    isSubmitting,
    currentStep,
    configuration,
    isConfigEdited,
    isYamlValid,
    showOverwriteConfirm,
    selectedWorkflows,
    engineCatalogDetails,
    isEngineCatalogLoading,
    isEngineCatalogError,
    hasConfig,
    hasNoEnabledSteps,
    canProceedToReview,
    canStart,
    configValidationRef,
    setCurrentStep,
    setSelectedAgentID,
    setInputSource,
    handleConfigSync,
    handleManualConfigChange,
    handleResetWorkflowConfig,
    handleScanWorkflowNameChange,
    handleOverwriteConfirm,
    handleOverwriteCancel,
    handleYamlValidationChange,
    handleInitiate,
    handleOpenChange,
    setShowOverwriteConfirm,
  } = useInitiateScanDialogState({
    open,
    organizationId,
    organizationIds,
    targetId,
    targetIds,
    onOpenChange,
    onSuccess,
    locale,
    tToast,
  })

  const steps = [
    { id: 1, title: t("steps.selectWorkflow") },
    { id: 2, title: t("steps.engineConfig") },
  ]

  const configuredEngineCount = React.useMemo(
    () => configuration.trim()
      ? countConfiguredWorkflowEngines(configuration)
      : countWorkflowConfiguredEngines(selectedWorkflows),
    [configuration, selectedWorkflows]
  )
  const selectedWorkflowDisplayName = React.useMemo(() => {
    const selectedWorkflowName = selectedWorkflowNames[0]
    const selectedWorkflow = workflows?.find((workflow) => workflow.name === selectedWorkflowName)
    return selectedWorkflow?.displayName || selectedWorkflow?.name || selectedWorkflowName
  }, [selectedWorkflowNames, workflows])
  const stepProgress = getScanStepProgress(currentStep, steps.length)

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

  const description = bulkScopeLabel ? (
    <>{bulkScopeLabel}</>
  ) : targetName ? (
    <>{t("targetDesc")} <span className="font-medium text-foreground">{targetName}</span></>
  ) : (
    <>{t("orgDesc")} <span className="font-medium text-foreground">{organization?.name}</span></>
  )

  return (
    <>
      <Drawer open={open} onOpenChange={handleOpenChange} swipeDirection="right">
        <DrawerContent
          className={cn(scanWorkbenchDrawerContentClassName, "data-[swipe-direction=right]:sm:max-w-[640px]")}
        >
          <form className="flex h-full min-h-0 w-full flex-col">
            <DrawerHeader className="shrink-0 bg-card px-5 pt-4 text-left sm:px-6">
              <EdgePanelHeader
                variant="workbench"
                leading={<Play className="size-7" />}
                title={<DrawerTitle className={textRole.sectionTitle}>{t("title")}</DrawerTitle>}
                description={(
                  <DrawerDescription className={cn("max-w-2xl", textRole.helperText)}>
                    {description}
                  </DrawerDescription>
                )}
                actions={(
                  <>
                    <span className={cn("hidden sm:inline", textRole.helperText)}>
                      {t("stepIndicator", { current: currentStep, total: steps.length })}
                    </span>
                    <DrawerClose
                      render={(
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon-sm"
                          aria-label={tActions("close")}
                          disabled={isSubmitting}
                        />
                      )}
                    >
                      <semanticIcons.action.cancel />
                    </DrawerClose>
                  </>
                )}
                progress={<InitiateScanStepHeader steps={steps} currentStep={currentStep} progress={stepProgress} />}
              />
            </DrawerHeader>

            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              <section
                className={cn(
                  "min-h-0 flex-1 px-5 sm:px-6",
                  currentStep === 2
                    ? "flex flex-col overflow-hidden py-3"
                    : "overflow-y-auto py-5"
                )}
              >
                {currentStep === 1 && (
                  <>
                    <InitiateScanWorkflowSelection
                      t={t}
                      selectedWorkflowNames={selectedWorkflowNames}
                      workflows={workflows}
                      isLoadingWorkflows={isLoadingWorkflows}
                      isWorkflowsError={isWorkflowsError}
                      isSubmitting={isSubmitting}
                      onWorkflowNamesChange={handleScanWorkflowNameChange}
                    />
                    <InitiateScanExecutionOptions
                      t={t}
                      inputSource={inputSource}
                      selectedAgentID={selectedAgentID}
                      isSubmitting={isSubmitting}
                      onInputSourceChange={setInputSource}
                      onAgentChange={setSelectedAgentID}
                    />
                  </>
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
                    selectedWorkflowNames={selectedWorkflowNames}
                    locale={locale}
                    engineCatalogDetails={engineCatalogDetails}
                    isEngineCatalogLoading={isEngineCatalogLoading}
                    isEngineCatalogError={isEngineCatalogError}
                    isSubmitting={isSubmitting}
                    onConfigSync={handleConfigSync}
                    onConfigChange={handleManualConfigChange}
                    onResetConfig={handleResetWorkflowConfig}
                    onYamlValidationChange={handleYamlValidationChange}
                    configValidationRef={configValidationRef}
                  />
                )}
              </section>
            </div>

            <div className="shrink-0 border-t bg-card px-5 py-4 sm:px-6">
              <InitiateScanFooter
                t={t}
                currentStep={currentStep}
                selectMode={selectMode}
                selectedWorkflowNames={selectedWorkflowNames}
                selectedWorkflowDisplayName={selectedWorkflowDisplayName}
                canProceedToReview={canProceedToReview}
                canStart={canStart}
                hasNoEnabledSteps={hasNoEnabledSteps}
                configuredEngineCount={configuredEngineCount}
                isSubmitting={isSubmitting}
                onBack={() => setCurrentStep((prev) => Math.max(1, prev - 1))}
                onNext={() => setCurrentStep(2)}
                onStart={handleInitiate}
              />
            </div>
          </form>
        </DrawerContent>
      </Drawer>

      <InitiateScanOverwriteDialog
        t={t}
        open={showOverwriteConfirm}
        onOpenChange={handleOverwriteDialogChange}
        onCancel={handleOverwriteCancel}
        onConfirm={handleOverwriteConfirm}
      />
    </>
  )
}

interface BulkInitiateScanDrawerProps {
  organizationIds?: number[]
  targetIds?: number[]
  scopeLabel: string
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

export function BulkInitiateScanDrawer({
  organizationIds,
  targetIds,
  scopeLabel,
  open,
  onOpenChange,
  onSuccess,
}: BulkInitiateScanDrawerProps) {
  return (
    <InitiateScanDrawer
      organizationIds={organizationIds}
      targetIds={targetIds}
      bulkScopeLabel={scopeLabel}
      open={open}
      onOpenChange={onOpenChange}
      onSuccess={onSuccess}
    />
  )
}
