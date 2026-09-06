"use client"

import React from "react"
import { FormDrawer } from "@/components/shared/form-drawer"
import { useEditScheduledScanDialogState } from "@/components/scan/scheduled/edit-scheduled-scan-dialog-state"
import {
  InitiateScanConfigStep,
  InitiateScanExecutionOptions,
  InitiateScanWorkflowSelection,
} from "@/components/scan/initiate-scan-dialog-sections"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  EditScheduledScanNameField,
  EditScheduledScanTargetSection,
  EditScheduledScanCronSection,
  EditScheduledScanDialogFooter,
} from "@/components/scan/scheduled/edit-scheduled-scan-dialog-sections"
import { useLocale, useTranslations } from "next-intl"
import type { Locale } from "@/i18n/config"
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
  const tInitiate = useTranslations("scan.initiate")
  const locale = useLocale() as Locale
  const [activeTab, setActiveTab] = React.useState("basic")
  const {
    cronPresets,
    displayName,
    setDisplayName,
    scanWorkflow,
    selectedTargetId,
    selectedAgentID,
    inputSource,
    cronExpression,
    setCronExpression,
    configuration,
    isConfigEdited,
    isYamlValid,
    isConfigurationSaveBlocked,
    selectedWorkflows,
    engineCatalogDetails,
    isEngineCatalogLoading,
    isEngineCatalogError,
    configValidationRef,
    targets,
    workflows,
    isPending,
    isLoadingWorkflows,
    isWorkflowsError,
    isWorkflowConfigLoading,
    handleWorkflowNamesChange,
    handleConfigSync,
    handleManualConfigChange,
    handleResetWorkflowConfig,
    handleYamlValidationChange,
    handleTargetSelect,
    setSelectedAgentID,
    setInputSource,
    handleSubmit,
  } = useEditScheduledScanDialogState({
    open,
    scheduledScan,
    locale,
    onOpenChange,
    onSuccess,
    t,
  })

  const handleFormSubmit = React.useCallback(
    (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()
      handleSubmit()
    },
    [handleSubmit]
  )

  React.useEffect(() => {
    if (open) setActiveTab("basic")
  }, [open, scheduledScan?.id])

  if (!scheduledScan) return null

  return (
    <FormDrawer
      open={open}
      onOpenChange={onOpenChange}
      title={t("editTitle")}
      description={t("editDesc")}
      closeDisabled={isPending}
      bodyClassName="flex min-h-0 flex-col overflow-hidden"
      formProps={{ onSubmit: handleFormSubmit }}
      footer={(
        <EditScheduledScanDialogFooter
          t={t}
          isPending={isPending || isWorkflowConfigLoading || isConfigurationSaveBlocked}
        />
      )}
    >
      <Tabs value={activeTab} onValueChange={setActiveTab} className="flex min-h-0 flex-1 flex-col gap-4">
        <TabsList variant="content" className="w-full justify-start border-b" aria-label={t("editTabs.label")}>
          <TabsTrigger value="basic" variant="content">
            {t("editTabs.basicInfo")}
          </TabsTrigger>
          <TabsTrigger value="configuration" variant="content">
            {t("editTabs.scanConfiguration")}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="basic" keepMounted className="min-h-0 overflow-y-auto pr-1">
          <div className="grid gap-4">
            <EditScheduledScanNameField
              t={t}
              name={displayName}
              onNameChange={setDisplayName}
            />

            <InitiateScanWorkflowSelection
              t={tInitiate}
              workflows={workflows}
              selectedWorkflowNames={scanWorkflow ? [scanWorkflow] : []}
              isLoadingWorkflows={isLoadingWorkflows}
              isWorkflowsError={isWorkflowsError}
              isSubmitting={isPending || isWorkflowConfigLoading}
              onWorkflowNamesChange={handleWorkflowNamesChange}
            />

            <InitiateScanExecutionOptions
              t={tInitiate}
              inputSource={inputSource}
              selectedAgentID={selectedAgentID}
              isSubmitting={isPending || isWorkflowConfigLoading}
              onInputSourceChange={setInputSource}
              onAgentChange={setSelectedAgentID}
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
              disabled={isPending}
            />
          </div>
        </TabsContent>

        <TabsContent value="configuration" keepMounted className="flex min-h-0 flex-1 flex-col overflow-hidden">
          <InitiateScanConfigStep
            t={tInitiate}
            configuration={configuration}
            selectedWorkflows={selectedWorkflows}
            isConfigEdited={isConfigEdited}
            isYamlValid={isYamlValid}
            hasConfig={configuration.trim().length > 0}
            selectMode="custom"
            selectedWorkflowNames={scanWorkflow ? [scanWorkflow] : []}
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
        </TabsContent>
      </Tabs>
    </FormDrawer>
  )
}
