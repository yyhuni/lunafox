"use client"

import React from "react"
import { Plus } from "@/components/icons"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Form } from "@/components/ui/form"

import { useAddOrganizationDialogState } from "@/components/organization/add-organization-dialog-state"
import {
  AddOrganizationHeader,
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
    targetValidation,
    isFormValid,
    isSubmitting,
    createOrganization,
    handleTextareaScroll,
    onSubmit,
  } = useAddOrganizationDialogState({
    onAdd,
    open: externalOpen,
    onOpenChange: externalOnOpenChange,
    tValidation,
  })

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      {externalOpen === undefined && (
        <DialogTrigger asChild>
          <Button size="sm">
            <Plus />
            {t("addButton")}
          </Button>
        </DialogTrigger>
      )}

      <DialogContent className="sm:max-w-[650px] max-h-[90vh] overflow-y-auto">
        <AddOrganizationHeader t={t} />

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <div className="grid gap-4 py-4">
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
                targetValidation={targetValidation}
                name="targets"
              />
            </div>

            <AddOrganizationFooter
              t={t}
              isSubmitting={isSubmitting}
              isFormValid={isFormValid}
              createPending={createOrganization.isPending}
              onCancel={() => handleOpenChange(false)}
            />
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
