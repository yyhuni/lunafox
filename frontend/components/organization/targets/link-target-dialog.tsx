"use client"

import React from "react"
import { semanticIcons } from "@/components/icons"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import { Form } from "@/components/ui/form"
import { FormDrawer, FormDrawerPanel } from "@/components/shared/form-drawer"

import type { BatchCreateResponse } from "@/types/api-response.types"
import { useLinkTargetDialogState } from "@/components/organization/targets/link-target-dialog-state"
import {
  LinkTargetDialogFooter,
  LinkTargetInputSection,
  LinkTargetOrganizationSection,
} from "@/components/organization/targets/link-target-dialog-sections"

// Component attribute type definition
interface LinkTargetDialogProps {
  organizationId: number                                     // Organization ID (fixed, cannot be modified)
  organizationName: string                                   // Organization name
  onAdd?: (result: BatchCreateResponse) => void              // Add a success callback to return batch created statistics
  open?: boolean                                             // External control dialog box switch status
  onOpenChange?: (open: boolean) => void                     // External control dialog box switch callback
}

interface LinkTargetFormPanelProps {
  organizationId: number
  organizationName: string
  onAdd?: (result: BatchCreateResponse) => void
  onOpenChange: (open: boolean) => void
}

interface LinkTargetFormSurfaceProps {
  organizationId: number
  organizationName: string
  onAdd?: (result: BatchCreateResponse) => void
  open?: boolean
  onOpenChange?: (open: boolean) => void
  presentation: "drawer" | "panel"
}

const AddIcon = semanticIcons.action.add
const TargetIcon = semanticIcons.concept.target

/**
 * Relevant target dialog component (using React Query)
 * 
 * Features:
 * 1. Enter targets in batches and associate them with organizations
 * 2. Automatically create non-existing targets
 * 3. Automatically manage submission status
 * 4. Automatic error handling and success prompts
 * 5. Fixed organization ID and cannot be modified
 */
export function LinkTargetDialog({ 
  organizationId,
  organizationName,
  onAdd,
  open: externalOpen, 
  onOpenChange: externalOnOpenChange,
}: LinkTargetDialogProps) {
  return (
    <LinkTargetFormSurface
      presentation="drawer"
      organizationId={organizationId}
      organizationName={organizationName}
      onAdd={onAdd}
      open={externalOpen}
      onOpenChange={externalOnOpenChange}
    />
  )
}

export function LinkTargetFormPanel({
  organizationId,
  organizationName,
  onAdd,
  onOpenChange,
}: LinkTargetFormPanelProps) {
  return (
    <LinkTargetFormSurface
      presentation="panel"
      organizationId={organizationId}
      organizationName={organizationName}
      onAdd={onAdd}
      open
      onOpenChange={onOpenChange}
    />
  )
}

function LinkTargetFormSurface({
  organizationId,
  organizationName,
  onAdd,
  open: externalOpen,
  onOpenChange: externalOnOpenChange,
  presentation,
}: LinkTargetFormSurfaceProps) {
  const t = useTranslations("organization.linkTarget")
  const tTarget = useTranslations("target")
  const {
    form,
    open,
    handleOpenChange,
    lineNumbersRef,
    textareaRef,
    targetValidation,
    isFormValid,
    handleTextareaScroll,
    batchCreateTargets,
    onSubmit,
  } = useLinkTargetDialogState({
    organizationId,
    onAdd,
    open: externalOpen,
    onOpenChange: externalOnOpenChange,
    t,
  })

  const trigger = externalOpen === undefined ? (
    <Button size="sm" variant="secondary">
      <AddIcon />
      {tTarget("addTarget")}
    </Button>
  ) : undefined

  const footer = (
    <LinkTargetDialogFooter
      t={t}
      isPending={batchCreateTargets.isPending}
      isFormValid={isFormValid}
    />
  )

  const fields = (
    <>
      <LinkTargetInputSection
        t={t}
        formControl={form.control}
        name="targets"
        lineNumbersRef={lineNumbersRef}
        textareaRef={textareaRef}
        onScroll={handleTextareaScroll}
        isPending={batchCreateTargets.isPending}
        targetValidation={targetValidation}
      />

      <LinkTargetOrganizationSection organizationName={organizationName} t={t} />
    </>
  )

  if (presentation === "panel") {
    return (
      <Form {...form}>
        <FormDrawerPanel
          title={t("title")}
          description={t("description", { name: organizationName })}
          icon={<TargetIcon className="size-4" />}
          closeDisabled={batchCreateTargets.isPending}
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
        open={open}
        onOpenChange={handleOpenChange}
        trigger={trigger}
        title={t("title")}
        description={t("description", { name: organizationName })}
        icon={<TargetIcon className="size-4" />}
        closeDisabled={batchCreateTargets.isPending}
        formProps={{ onSubmit: form.handleSubmit(onSubmit) }}
        footer={footer}
      >
        {fields}
      </FormDrawer>
    </Form>
  )
}
