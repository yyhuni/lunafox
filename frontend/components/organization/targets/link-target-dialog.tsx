"use client"

import React from "react"
import { Plus } from "@/components/icons"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogTrigger } from "@/components/ui/dialog"
import { Form } from "@/components/ui/form"

import type { BatchCreateResponse } from "@/types/api-response.types"
import { useLinkTargetDialogState } from "@/components/organization/targets/link-target-dialog-state"
import {
  LinkTargetDialogFooter,
  LinkTargetDialogHeader,
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
  const t = useTranslations("organization.linkTarget")
  const tCommon = useTranslations("common")
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

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      {/* Trigger button - only shown when not externally controlled */}
      {externalOpen === undefined && (
        <DialogTrigger asChild>
          <Button size="sm" variant="secondary">
            <Plus />
            {tTarget("addTarget")}
          </Button>
        </DialogTrigger>
      )}
      
      {/* Dialog content */}
      <DialogContent className="sm:max-w-[650px] max-h-[90vh] overflow-y-auto">
        <LinkTargetDialogHeader organizationName={organizationName} t={t} />
        
        {/* form */}
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <div className="grid gap-4 py-4">
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
            </div>
          
          {/* Button at bottom of dialog box */}
          <LinkTargetDialogFooter
            tCommon={tCommon}
            t={t}
            onCancel={() => handleOpenChange(false)}
            isPending={batchCreateTargets.isPending}
            isFormValid={isFormValid}
          />
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
