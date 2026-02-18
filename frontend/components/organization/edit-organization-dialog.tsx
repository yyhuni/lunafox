"use client"

import React from "react"
import { useTranslations } from "next-intl"

import { Dialog, DialogContent } from "@/components/ui/dialog"
import { Form } from "@/components/ui/form"

import { useEditOrganizationDialogState } from "@/components/organization/edit-organization-dialog-state"
import {
  EditOrganizationHeader,
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

export function EditOrganizationDialog({
  organization,
  open,
  onOpenChange,
  onEdit,
}: EditOrganizationDialogProps) {
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

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <EditOrganizationHeader t={t} />

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <div className="grid gap-4 py-4">
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
            </div>

            <EditOrganizationFooter
              t={t}
              isUpdating={isUpdating}
              isFormValid={isFormValid}
              hasChanges={hasChanges}
              onCancel={() => handleOpenChange(false)}
              onReset={handleReset}
            />
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
