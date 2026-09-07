"use client"

import React from "react"
import { semanticIcons } from "@/components/icons"
import { useTranslations } from "next-intl"

import { Form } from "@/components/ui/form"
import { FormDrawer, FormDrawerPanel } from "@/components/shared/form-drawer"

import { useEditOrganizationDialogState } from "@/components/organization/edit-organization-dialog-state"
import {
  EditOrganizationNameField,
  EditOrganizationDescriptionField,
  EditOrganizationChangesNotice,
  EditOrganizationFooter,
} from "@/components/organization/edit-organization-dialog-sections"

import type { Organization } from "@/types/organization.types"

interface EditOrganizationDialogProps {
  organization: Organization
  open: boolean
  onOpenChange: (open: boolean) => void
  onEdit: (organization: Organization) => void
}

interface EditOrganizationFormPanelProps {
  organization: Organization
  onOpenChange: (open: boolean) => void
  onEdit: (organization: Organization) => void
}

interface EditOrganizationFormSurfaceProps extends EditOrganizationFormPanelProps {
  presentation: "drawer" | "panel"
  open?: boolean
}

const OrganizationIcon = semanticIcons.concept.organization

export function EditOrganizationDialog({
  organization,
  open,
  onOpenChange,
  onEdit,
}: EditOrganizationDialogProps) {
  return (
    <EditOrganizationFormSurface
      presentation="drawer"
      organization={organization}
      open={open}
      onOpenChange={onOpenChange}
      onEdit={onEdit}
    />
  )
}

export function EditOrganizationFormPanel({
  organization,
  onOpenChange,
  onEdit,
}: EditOrganizationFormPanelProps) {
  return (
    <EditOrganizationFormSurface
      presentation="panel"
      organization={organization}
      onOpenChange={onOpenChange}
      onEdit={onEdit}
    />
  )
}

function EditOrganizationFormSurface({
  organization,
  open,
  onOpenChange,
  onEdit,
  presentation,
}: EditOrganizationFormSurfaceProps) {
  const t = useTranslations("organization.dialog")
  const tValidation = useTranslations("organization.validation")

  const {
    form,
    hasChanges,
    isFormValid,
    isUpdating,
    handleOpenChange,
    handleReset,
    onSubmit,
  } = useEditOrganizationDialogState({
    organization,
    onOpenChange,
    onEdit,
    tValidation,
  })

  const footer = (
    <EditOrganizationFooter
      t={t}
      isUpdating={isUpdating}
      isFormValid={isFormValid}
      hasChanges={hasChanges}
      onReset={handleReset}
    />
  )

  const fields = (
    <>
      <EditOrganizationNameField
        t={t}
        formControl={form.control}
        isSubmitting={isUpdating}
        name="name"
      />
      <EditOrganizationDescriptionField
        t={t}
        formControl={form.control}
        isSubmitting={isUpdating}
        name="description"
      />
      {hasChanges && <EditOrganizationChangesNotice t={t} />}
    </>
  )

  if (presentation === "panel") {
    return (
      <Form {...form}>
        <FormDrawerPanel
          title={t("editTitle")}
          icon={<OrganizationIcon className="size-4" />}
          closeDisabled={isUpdating}
          onClose={() => handleOpenChange(false)}
          formProps={{ onSubmit: form.handleSubmit(onSubmit) }}
          footer={footer}
        >
          {fields}
        </FormDrawerPanel>
      </Form>
    )
  }

  return (
    <Form {...form}>
      <FormDrawer
        open={Boolean(open)}
        onOpenChange={handleOpenChange}
        title={t("editTitle")}
        icon={<OrganizationIcon className="size-4" />}
        closeDisabled={isUpdating}
        formProps={{ onSubmit: form.handleSubmit(onSubmit) }}
        footer={footer}
      >
        {fields}
      </FormDrawer>
    </Form>
  )
}
