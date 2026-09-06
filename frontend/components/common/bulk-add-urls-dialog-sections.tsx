"use client"

import React from "react"
import { Plus, semanticIcons } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { BulkLineValidationInput, type BulkLineValidationIssue } from "@/components/common/bulk-line-validation-input"
import type { BulkAddUrlLineIssue, BulkAddUrlLineIssueType } from "@/components/common/bulk-add-urls-dialog-state"
import type { URLValidationErrorCode } from "@/lib/url-validator"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface BulkAddUrlsDialogHeaderProps {
  title: string
  description: string
}

export function BulkAddUrlsDialogHeader({ title, description }: BulkAddUrlsDialogHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle className="flex items-center space-x-2">
        <semanticIcons.concept.endpoint className="h-5 w-5" />
        <span>{title}</span>
      </DialogTitle>
      <DialogDescription>{description}</DialogDescription>
    </DialogHeader>
  )
}

interface BulkAddUrlsInputProps {
  tUrl: TranslationFn
  placeholder: string
  inputText: string
  lineNumbersRef: React.RefObject<HTMLDivElement | null>
  textareaRef: React.RefObject<HTMLTextAreaElement | null>
  onInputChange: (value: string) => void
  onScroll: (event: React.UIEvent<HTMLTextAreaElement>) => void
  isPending: boolean
  lineCount: number
  validationResult: {
    validCount: number
    invalidCount: number
    duplicateCount: number
    mismatchedCount: number
    blockingIssueCount: number
    advisoryIssueCount: number
    invalidItems: Array<{ lineNumber: number; url: string; error?: string; errorCode?: URLValidationErrorCode }>
    duplicateItems: Array<{ lineNumber: number; url: string; duplicateOfLine: number }>
    mismatchedItems: Array<{ lineNumber: number; url: string }>
    lineIssues: BulkAddUrlLineIssue[]
  } | null
}

function getUrlValidationErrorMessage(tUrl: TranslationFn, errorCode?: URLValidationErrorCode): string {
  return tUrl(`errors.${errorCode || "unknown"}`)
}

function getIssueBadgeLabel(tUrl: TranslationFn, type: BulkAddUrlLineIssueType): string {
  if (type === "duplicate") return tUrl("duplicateBadge")
  if (type === "mismatch") return tUrl("mismatchBadge")
  return tUrl("invalidBadge")
}

function getIssueMessage(tUrl: TranslationFn, issue: BulkAddUrlLineIssue): string {
  if (issue.type === "duplicate") {
    return tUrl("duplicateIssue", { line: issue.lineNumber })
  }
  if (issue.type === "mismatch") {
    return tUrl("mismatchIssue", { line: issue.lineNumber })
  }
  return tUrl("invalidIssue", {
    line: issue.lineNumber,
    error: getUrlValidationErrorMessage(tUrl, issue.errorCode),
  })
}

export function BulkAddUrlsInput({
  tUrl,
  placeholder,
  inputText,
  lineNumbersRef,
  textareaRef,
  onInputChange,
  onScroll,
  isPending,
  lineCount,
  validationResult,
}: BulkAddUrlsInputProps) {
  const lineIssues = React.useMemo<BulkLineValidationIssue[]>(() => (
    validationResult?.lineIssues.map((issue) => ({
      id: `${issue.type}-${issue.lineNumber}`,
      lineNumber: issue.lineNumber,
      tone: issue.tone,
      badgeLabel: getIssueBadgeLabel(tUrl, issue.type),
      message: getIssueMessage(tUrl, issue),
    })) ?? []
  ), [tUrl, validationResult])

  return (
    <BulkLineValidationInput
      id="urls"
      name="urls"
      label={tUrl("label")}
      required
      placeholder={placeholder}
      value={inputText}
      lineCount={lineCount}
      lineNumbersRef={lineNumbersRef}
      textareaRef={textareaRef}
      onValueChange={onInputChange}
      onScroll={onScroll}
      disabled={isPending}
      helper={tUrl("helper")}
      example={tUrl("example")}
      emptySummary={tUrl("emptySummary")}
      validSummary={validationResult ? tUrl("validSummary", { count: validationResult.validCount }) : ""}
      blockingSummary={validationResult ? tUrl("blockingSummary", { count: validationResult.blockingIssueCount }) : ""}
      advisorySummary={validationResult ? tUrl("advisorySummary", { count: validationResult.advisoryIssueCount }) : undefined}
      collapseDetails={tUrl("collapseDetails")}
      expandDetails={tUrl("expandDetails")}
      validationResult={validationResult ? { ...validationResult, lineIssues } : null}
    />
  )
}

interface BulkAddUrlsFooterProps {
  tBulkAdd: TranslationFn
  tUrl: TranslationFn
  isPending: boolean
  isFormValid: boolean
}

export function BulkAddUrlsFooter({
  tBulkAdd,
  tUrl,
  isPending,
  isFormValid,
}: BulkAddUrlsFooterProps) {
  return (
    <DialogFooter>
      <Button
        type="submit"
        disabled={isPending || !isFormValid}
        loading={isPending}
        loadingLabel={tUrl("creating")}
      >
        <>
          <Plus className="h-4 w-4" />
          {tBulkAdd("bulkAdd")}
        </>
      </Button>
    </DialogFooter>
  )
}
