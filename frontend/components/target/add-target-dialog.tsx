"use client"

import React from "react"
import { Plus, semanticIcons } from "@/components/icons"
import { useTranslations } from "next-intl"

// Import UI components
import { Button } from "@/components/ui/button"
import { FormDrawer } from "@/components/shared/form-drawer"
import { useAddTargetDialogState } from "@/components/target/add-target-dialog-state"
import { scanWorkbenchDrawerContentClassName } from "@/lib/ui/overlay-styles"
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

const TargetIcon = semanticIcons.concept.target

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
  const {
    open,
    handleOpenChange,
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
    filteredOrganizations,
    orgSearchQuery,
    setOrgSearchQuery,
    orgPageSize,
    setOrgPageSize,
    organizationTotalCount,
    organizationPaginationNavigation,
    onFirstOrgPage,
    onPreviousOrgPage,
    onNextOrgPage,
    handleToggleOrganization,
    handleClearOrganizations,
    batchCreateTargets,
  } = useAddTargetDialogState({
    onAdd,
    externalOpen,
    externalOnOpenChange,
    prefetchEnabled,
    t,
  })

  const trigger = externalOpen === undefined ? (
    <Button size="sm">
      <Plus />
      {t("addTarget")}
    </Button>
  ) : undefined

  return (
    <FormDrawer
      open={open}
      onOpenChange={handleOpenChange}
      trigger={trigger}
      title={t("addTitle")}
      description={t("addDesc")}
      icon={<TargetIcon className="size-4" />}
      className={scanWorkbenchDrawerContentClassName}
      bodyClassName="gap-6 px-5 py-5 sm:px-6 sm:py-6"
      closeDisabled={batchCreateTargets.isPending}
      formProps={{ onSubmit: handleSubmit }}
      footer={(
        <AddTargetDialogFooter
          tDialog={t}
          isPending={batchCreateTargets.isPending}
          isFormValid={isFormValid}
        />
      )}
    >
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
        isLoading={isLoadingOrganizations}
        formOrganizationIds={formData.organizationIds}
        organizations={filteredOrganizations}
        organizationTotalCount={organizationTotalCount}
        organizationPaginationNavigation={organizationPaginationNavigation}
        onFirstOrgPage={onFirstOrgPage}
        orgSearchQuery={orgSearchQuery}
        setOrgSearchQuery={setOrgSearchQuery}
        orgPageSize={orgPageSize}
        setOrgPageSize={setOrgPageSize}
        onPreviousOrgPage={onPreviousOrgPage}
        onNextOrgPage={onNextOrgPage}
        onToggleOrganization={handleToggleOrganization}
        onClearOrganizations={handleClearOrganizations}
        isPending={batchCreateTargets.isPending}
      />
    </FormDrawer>
  )
}
