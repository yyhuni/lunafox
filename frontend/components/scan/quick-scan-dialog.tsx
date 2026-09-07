"use client"

import * as React from "react"
import { useLocale, useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import { Drawer, DrawerContent, DrawerTrigger } from "@/components/ui/drawer"
import {
  InitiateScanConfigStep,
  InitiateScanExecutionOptions,
  InitiateScanFooter,
  InitiateScanOverwriteDialog,
  InitiateScanStepHeader,
  InitiateScanWorkflowSelection,
  countConfiguredWorkflowEngines,
  countWorkflowConfiguredEngines,
  getScanStepProgress,
} from "@/components/scan/initiate-scan-dialog-sections"
import {
  QuickScanFooter,
  QuickScanHeader,
  QuickScanTargetStep,
} from "@/components/scan/quick-scan-dialog-sections"
import { useQuickScanDialogState } from "@/components/scan/quick-scan-dialog-state"
import { scanWorkbenchDrawerContentClassName } from "@/lib/ui/overlay-styles"
import { cn } from "@/lib/utils"
import type { Locale } from "@/i18n/config"

interface QuickScanDialogProps {
  trigger?: React.ReactElement
}

export function QuickScanDialog({ trigger }: QuickScanDialogProps) {
  const t = useTranslations("quickScan")
  const tTargetDialog = useTranslations("target.dialog")
  const tInitiate = useTranslations("scan.initiate")
  const tActions = useTranslations("common.actions")
  const locale = useLocale() as Locale
  const {
    open,
    handleClose,
    isSubmitting,
    step,
    targetInput,
    setTargetInput,
    selectedWorkflowNames,
    selectedAgentID,
    inputSource,
    selectMode,
    configuration,
    isConfigEdited,
    isYamlValid,
    showOverwriteConfirm,
    setShowOverwriteConfirm,
    lineNumbersRef,
    textareaRef,
    handleTextareaScroll,
    validInputs,
    invalidInputs,
    workflows,
    isLoadingWorkflows,
    isWorkflowsError,
    selectedWorkflows,
    engineCatalogDetails,
    isEngineCatalogLoading,
    isEngineCatalogError,
    hasConfig,
    hasNoEnabledSteps,
    handleConfigSync,
    handleManualConfigChange,
    handleResetWorkflowConfig,
    handleScanWorkflowNameChange,
    handleOverwriteConfirm,
    handleOverwriteCancel,
    handleYamlValidationChange,
    setSelectedAgentID,
    setInputSource,
    canProceedToStep2,
    canProceedToStep3,
    canSubmit,
    handleNext,
    handleBack,
    handleSubmit,
    totalSteps,
  } = useQuickScanDialogState({ t, locale })

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
  const steps = [
    { id: 1, title: t("scanTargets") },
    { id: 2, title: tInitiate("steps.selectWorkflow") },
    { id: 3, title: tInitiate("steps.engineConfig") },
  ]
  const stepProgress = getScanStepProgress(step, totalSteps)

  const defaultTrigger = (
    <Button
      data-slot="quick-scan-trigger"
      variant="outline"
      size="sm"
      className="bg-secondary/50 border-0 group overflow-hidden px-4 relative"
    >
      <div className="absolute border border-highlight/0 duration-300 group-hover:border-highlight/100 group-hover:scale-100 inset-0 scale-95 transition-colors" />
      <div className="absolute bg-highlight duration-300 group-hover:opacity-100 h-1 left-0 opacity-0 top-0 transition-opacity w-1" />
      <div className="absolute bg-highlight duration-300 group-hover:opacity-100 h-1 opacity-0 right-0 top-0 transition-opacity w-1" />
      <div className="absolute bg-highlight bottom-0 duration-300 group-hover:opacity-100 h-1 left-0 opacity-0 transition-opacity w-1" />
      <div className="absolute bg-highlight bottom-0 duration-300 group-hover:opacity-100 h-1 opacity-0 right-0 transition-opacity w-1" />
      <span className="absolute duration-300 font-mono group-hover:opacity-100 left-2 mr-2 opacity-0 text-highlight text-xs transition-opacity">
        {">"}
      </span>
      <div className="duration-300 flex group-hover:text-highlight items-center transition-colors">
        <span className="font-medium">{t("title")}</span>
      </div>
    </Button>
  )

  return (
    <Drawer open={open} onOpenChange={handleClose} swipeDirection="right">
      <DrawerTrigger render={trigger ?? defaultTrigger} />
      <DrawerContent
        data-slot="quick-scan-drawer"
        className={cn(
          scanWorkbenchDrawerContentClassName,
          "data-[swipe-direction=right]:sm:max-w-[640px]"
        )}
      >
        <form className="flex h-full min-h-0 w-full flex-col">
          <QuickScanHeader
            t={t}
            step={step}
            totalSteps={totalSteps}
            closeLabel={tActions("close")}
            isSubmitting={isSubmitting}
            stepHeader={<InitiateScanStepHeader steps={steps} currentStep={step} progress={stepProgress} />}
          />

          <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
            <section
              className={cn(
                "min-h-0 flex-1 px-5 sm:px-6",
                step === 3
                  ? "flex flex-col overflow-hidden py-3"
                  : "overflow-y-auto py-5"
              )}
            >
              {step === 1 && (
                <QuickScanTargetStep
                  tTargetDialog={tTargetDialog}
                  targetInput={targetInput}
                  validCount={validInputs.length}
                  onTargetChange={setTargetInput}
                  lineNumbersRef={lineNumbersRef}
                  textareaRef={textareaRef}
                  onScroll={handleTextareaScroll}
                  invalidInputs={invalidInputs}
                  isSubmitting={isSubmitting}
                />
              )}

              {step === 2 && (
                <>
                  <InitiateScanWorkflowSelection
                    t={tInitiate}
                    selectedWorkflowNames={selectedWorkflowNames}
                    workflows={workflows}
                    isLoadingWorkflows={isLoadingWorkflows}
                    isWorkflowsError={isWorkflowsError}
                    isSubmitting={isSubmitting}
                    onWorkflowNamesChange={handleScanWorkflowNameChange}
                  />
                  <InitiateScanExecutionOptions
                    t={tInitiate}
                    inputSource={inputSource}
                    selectedAgentID={selectedAgentID}
                    isSubmitting={isSubmitting}
                    onInputSourceChange={setInputSource}
                    onAgentChange={setSelectedAgentID}
                  />
                </>
              )}

              {step === 3 && (
                <InitiateScanConfigStep
                  t={tInitiate}
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
                />
              )}
            </section>
          </div>

          <div className="shrink-0 border-t bg-card px-5 py-4 sm:px-6">
            {step === 1 ? (
              <QuickScanFooter
                t={t}
                step={step}
                validCount={validInputs.length}
                invalidCount={invalidInputs.length}
                selectedScanWorkflowCount={0}
                isSubmitting={isSubmitting}
                canProceedToStep2={canProceedToStep2}
                canProceedToStep3={false}
                canSubmit={false}
                onBack={handleBack}
                onNext={handleNext}
                onSubmit={handleSubmit}
              />
            ) : (
              <InitiateScanFooter
                t={tInitiate}
                currentStep={step - 1}
                showBackButton={step > 1}
                selectMode={selectMode}
                selectedWorkflowNames={selectedWorkflowNames}
                selectedWorkflowDisplayName={selectedWorkflowDisplayName}
                canProceedToReview={canProceedToStep3}
                canStart={canSubmit}
                hasNoEnabledSteps={hasNoEnabledSteps}
                configuredEngineCount={configuredEngineCount}
                isSubmitting={isSubmitting}
                onBack={handleBack}
                onNext={handleNext}
                onStart={handleSubmit}
              />
            )}
          </div>
        </form>
      </DrawerContent>

      <InitiateScanOverwriteDialog
        t={tInitiate}
        open={showOverwriteConfirm}
        onOpenChange={setShowOverwriteConfirm}
        onCancel={handleOverwriteCancel}
        onConfirm={handleOverwriteConfirm}
      />
    </Drawer>
  )
}
