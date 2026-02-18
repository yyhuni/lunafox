"use client"

import React from "react"
import { Plus, Target as TargetIcon } from "@/components/icons"
import { useTranslations } from "next-intl"

// Import UI components
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { useAddTargetDialogState } from "@/components/target/add-target-dialog-state"
import { buildPaginationInfo, normalizePagination } from "@/hooks/_shared/pagination"
import {
  AddTargetDialogFooter,
  AddTargetInputSection,
  AddTargetOrganizationPicker,
} from "@/components/target/add-target-dialog-sections"

// Import React Query Hooks

// Component props type definition
interface AddTargetDialogProps {
  onAdd?: () => void                                             // Success callback after adding
  open?: boolean                                                 // External control for dialog open state
  onOpenChange?: (open: boolean) => void                         // External control for dialog open callback
  prefetchEnabled?: boolean                                      // Whether to prefetch organization list
}

/**
 * Add target dialog component (supports organization selection)
 * 
 * Features:
 * 1. Batch input targets
 * 2. Optional organization selection
 * 3. Auto-create non-existent targets
 * 4. Auto-manage submission state
 * 5. Auto error handling and success notifications
 */
export function AddTargetDialog({ 
  onAdd,
  open: externalOpen, 
  onOpenChange: externalOnOpenChange,
  prefetchEnabled,
}: AddTargetDialogProps) {
  const t = useTranslations("target.dialog")
  const tCommon = useTranslations("common.actions")
  const tPagination = useTranslations("common.pagination")
  const {
    open,
    handleOpenChange,
    orgPickerOpen,
    setOrgPickerOpen,
    handleOrgPickerOpenChange,
    formData,
    handleInputChange,
    handleSubmit,
    targetCount,
    invalidTargets,
    isFormValid,
    lineNumbersRef,
    textareaRef,
    handleTextareaScroll,
    isLoadingOrganizations,
    organizationsData,
    filteredOrganizations,
    selectedOrgName,
    orgSearchQuery,
    setOrgSearchQuery,
    orgPage,
    setOrgPage,
    orgPageSize,
    setOrgPageSize,
    pageSizeOptions,
    handleSelectOrganization,
    batchCreateTargets,
  } = useAddTargetDialogState({
    onAdd,
    externalOpen,
    externalOnOpenChange,
    prefetchEnabled,
    t,
  })

  const organizationPaginationInfo = organizationsData
    ? buildPaginationInfo({
      ...normalizePagination(
        organizationsData.pagination,
        orgPage,
        orgPageSize
      ),
      minTotalPages: 1,
    })
    : undefined

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      {/* Trigger button - only shown when not externally controlled */}
      {externalOpen === undefined && (
        <DialogTrigger asChild>
          <Button size="sm">
            <Plus />
            {t("addTarget")}
          </Button>
        </DialogTrigger>
      )}
      
      {/* Dialog content */}
      <DialogContent className="sm:max-w-[650px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center space-x-2">
            <TargetIcon />
            <span>{t("addTitle")}</span>
          </DialogTitle>
          <DialogDescription>
            {t("addDesc")}
          </DialogDescription>
        </DialogHeader>
        
        {/* Form */}
        <form onSubmit={handleSubmit}>
          <div className="grid gap-4 py-4">
            <AddTargetInputSection
              t={t}
              formTargets={formData.targets}
              targetCount={targetCount}
              invalidTargets={invalidTargets}
              lineNumbersRef={lineNumbersRef}
              textareaRef={textareaRef}
              onInputChange={(value) => handleInputChange("targets", value)}
              onScroll={handleTextareaScroll}
              isPending={batchCreateTargets.isPending}
            />

            <AddTargetOrganizationPicker
              t={t}
              tPagination={tPagination}
              isLoading={isLoadingOrganizations}
              orgPickerOpen={orgPickerOpen}
              handleOrgPickerOpenChange={handleOrgPickerOpenChange}
              formOrganizationId={formData.organizationId}
              selectedOrgName={selectedOrgName}
              organizations={filteredOrganizations}
              organizationPaginationInfo={organizationPaginationInfo}
              orgSearchQuery={orgSearchQuery}
              setOrgSearchQuery={setOrgSearchQuery}
              orgPage={orgPage}
              setOrgPage={setOrgPage}
              orgPageSize={orgPageSize}
              setOrgPageSize={setOrgPageSize}
              pageSizeOptions={pageSizeOptions}
              onSelectOrganization={handleSelectOrganization}
              isPending={batchCreateTargets.isPending}
              onOpenPicker={() => setOrgPickerOpen(true)}
            />
          </div>
          
          <AddTargetDialogFooter
            t={tCommon}
            isPending={batchCreateTargets.isPending}
            isFormValid={isFormValid}
            onCancel={() => handleOpenChange(false)}
          />
        </form>
      </DialogContent>
    </Dialog>
  )
}
