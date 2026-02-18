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
import { type TargetType } from "@/lib/url-validator"
import { useBulkAddUrlsDialogState, type AssetType } from "@/components/common/bulk-add-urls-dialog-state"
import {
  BulkAddUrlsDialogHeader,
  BulkAddUrlsFooter,
  BulkAddUrlsInput,
} from "@/components/common/bulk-add-urls-dialog-sections"

interface BulkAddUrlsDialogProps {
  targetId: number
  assetType: AssetType
  targetName?: string      // Target name (used for URL matching validation)
  targetType?: TargetType  // Target type (domain/ip/cidr)
  open?: boolean
  onOpenChange?: (open: boolean) => void
  onSuccess?: () => void
}

/**
 * Bulk add URLs dialog component
 * 
 * Supports three asset types: Endpoints, Websites, Directories.
 * Provides text input with line numbers, supports real-time validation and error hints.
 */
export function BulkAddUrlsDialog({
  targetId,
  assetType,
  targetName,
  targetType,
  open: externalOpen,
  onOpenChange: externalOnOpenChange,
  onSuccess,
}: BulkAddUrlsDialogProps) {
  const tBulkAdd = useTranslations("bulkAdd.common")
  const tUrl = useTranslations("bulkAdd.url")
  
  // Get translated labels based on asset type
  const labels = {
    title: tUrl(`${assetType}.title`),
    description: tUrl(`${assetType}.description`),
    placeholder: tUrl(`${assetType}.placeholder`),
  }
  
  const {
    open,
    handleOpenChange,
    inputText,
    validationResult,
    lineNumbersRef,
    textareaRef,
    mutation,
    handleInputChange,
    handleSubmit,
    handleTextareaScroll,
    lineCount,
    isFormValid,
  } = useBulkAddUrlsDialogState({
    targetId,
    assetType,
    targetName,
    targetType,
    open: externalOpen,
    onOpenChange: externalOnOpenChange,
    onSuccess,
    tBulkAdd,
  })

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      {externalOpen === undefined && (
        <DialogTrigger asChild>
          <Button size="sm" variant="outline">
            <Plus className="h-4 w-4" />
            {tBulkAdd("bulkAdd")}
          </Button>
        </DialogTrigger>
      )}

      <DialogContent className="sm:max-w-[650px] max-h-[90vh] overflow-y-auto">
        <BulkAddUrlsDialogHeader title={labels.title} description={labels.description} />

        <form onSubmit={handleSubmit}>
          <div className="grid gap-4 py-4">
            <BulkAddUrlsInput
              tUrl={tUrl}
              placeholder={labels.placeholder}
              inputText={inputText}
              lineNumbersRef={lineNumbersRef}
              textareaRef={textareaRef}
              onInputChange={handleInputChange}
              onScroll={handleTextareaScroll}
              isPending={mutation.isPending}
              lineCount={lineCount}
              validationResult={validationResult}
              targetName={targetName}
            />
          </div>

          <BulkAddUrlsFooter
            tBulkAdd={tBulkAdd}
            tUrl={tUrl}
            onCancel={() => handleOpenChange(false)}
            isPending={mutation.isPending}
            isFormValid={isFormValid}
          />
        </form>
      </DialogContent>
    </Dialog>
  )
}
