"use client"

import React from "react"
import { semanticIcons } from "@/components/icons"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import { Form } from "@/components/ui/form"
import { FormDrawer } from "@/components/shared/form-drawer"

import { useAddOrganizationDialogState } from "@/components/organization/add-organization-dialog-state"
import {
  AddOrganizationNameField,
  AddOrganizationDescriptionField,
  AddOrganizationTargetsField,
  AddOrganizationFooter,
} from "@/components/organization/add-organization-dialog-sections"

import type { Organization } from "@/types/organization.types"

interface AddOrganizationDialogProps {
  onAdd?: (organization: Organization) => void
  open?: boolean
  onOpenChange?: (open: boolean) => void
}

const AddIcon = semanticIcons.action.add
const OrganizationIcon = semanticIcons.concept.organization

export function AddOrganizationDialog({
  onAdd,
  open: externalOpen,
  onOpenChange: externalOnOpenChange,
}: AddOrganizationDialogProps) {
  const t = useTranslations("organization.dialog")
  const tValidation = useTranslations("organization.validation")

  const {
    form,
    open,
    handleOpenChange,
    lineNumbersRef,
    textareaRef,
    targetsExpanded,
    targetValidation,
    isFormValid,
    isSubmitting,
    createOrganization,
    handleTextareaScroll,
    handleToggleTargetsExpanded,
    onSubmit,
  } = useAddOrganizationDialogState({
    onAdd,
    open: externalOpen,
    onOpenChange: externalOnOpenChange,
    tValidation,
  })

  const trigger = externalOpen === undefined ? (
    <Button size="sm">
      <AddIcon />
      {t("addButton")}
    </Button>
  ) : undefined

  return (
    <Form {...form}>
      <FormDrawer
        open={open}
        onOpenChange={handleOpenChange}
        trigger={trigger}
        title={t("addTitle")}
        description={t("addDesc")}
        icon={<OrganizationIcon className="size-4" />}
        closeDisabled={isSubmitting}
        formProps={{ onSubmit: form.handleSubmit(onSubmit) }}
        footer={(
          <AddOrganizationFooter
            t={t}
            isSubmitting={isSubmitting}
            isFormValid={isFormValid}
            createPending={createOrganization.isPending}
          />
        )}
      >
        <AddOrganizationNameField
          t={t}
          formControl={form.control}
          isSubmitting={isSubmitting}
          name="name"
        />
        <AddOrganizationDescriptionField
          t={t}
          formControl={form.control}
          isSubmitting={isSubmitting}
          name="description"
        />
        <AddOrganizationTargetsField
          t={t}
          formControl={form.control}
          isSubmitting={isSubmitting}
          lineNumbersRef={lineNumbersRef}
          textareaRef={textareaRef}
          onScroll={handleTextareaScroll}
          isTargetsExpanded={targetsExpanded}
          onToggleTargetsExpanded={handleToggleTargetsExpanded}
          targetValidation={targetValidation}
          name="targets"
        />
      </FormDrawer>
    </Form>
  )
}
