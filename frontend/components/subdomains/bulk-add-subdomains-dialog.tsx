"use client"

import React from "react"
import { Plus } from "@/components/icons"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogTrigger } from "@/components/ui/dialog"
import { useBulkAddSubdomainsDialogState } from "@/components/subdomains/bulk-add-subdomains-dialog-state"
import {
  BulkAddSubdomainsFooter,
  BulkAddSubdomainsHeader,
  BulkAddSubdomainsInput,
} from "@/components/subdomains/bulk-add-subdomains-dialog-sections"

interface BulkAddSubdomainsDialogProps {
  targetId: number
  targetName?: string
  open?: boolean
  onOpenChange?: (open: boolean) => void
  onSuccess?: () => void
}

/**
 * Bulk add subdomains dialog component
 * 
 * Following the design pattern of AddTargetDialog, provides a text input with line numbers,
 * supporting real-time validation and error prompts.
 */
export function BulkAddSubdomainsDialog({
  targetId,
  targetName,
  open: externalOpen,
  onOpenChange: externalOnOpenChange,
  onSuccess,
}: BulkAddSubdomainsDialogProps) {
  const t = useTranslations("bulkAdd.subdomain")
  const tCommon = useTranslations("common.actions")
  const {
    open,
    handleOpenChange,
    inputText,
    validationResult,
    lineNumbersRef,
    textareaRef,
    bulkCreateSubdomains,
    handleInputChange,
    handleSubmit,
    handleTextareaScroll,
    lineCount,
    isFormValid,
  } = useBulkAddSubdomainsDialogState({
    targetId,
    open: externalOpen,
    onOpenChange: externalOnOpenChange,
    onSuccess,
    t,
  })

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      {externalOpen === undefined && (
        <DialogTrigger asChild>
          <Button size="sm" variant="outline">
            <Plus className="h-4 w-4" />
            {t("bulkAdd")}
          </Button>
        </DialogTrigger>
      )}

      <DialogContent className="sm:max-w-[650px] max-h-[90vh] overflow-y-auto">
        <BulkAddSubdomainsHeader
          title={t("title")}
          description={t("description")}
          targetName={targetName}
          t={t}
        />

        <form onSubmit={handleSubmit}>
          <div className="grid gap-4 py-4">
            <BulkAddSubdomainsInput
              t={t}
              inputText={inputText}
              placeholder={t("placeholder")}
              lineNumbersRef={lineNumbersRef}
              textareaRef={textareaRef}
              onInputChange={handleInputChange}
              onScroll={handleTextareaScroll}
              isPending={bulkCreateSubdomains.isPending}
              lineCount={lineCount}
              validationResult={validationResult}
            />
          </div>

          <BulkAddSubdomainsFooter
            tCommon={tCommon}
            t={t}
            onCancel={() => handleOpenChange(false)}
            isPending={bulkCreateSubdomains.isPending}
            isFormValid={isFormValid}
          />
        </form>
      </DialogContent>
    </Dialog>
  )
}
