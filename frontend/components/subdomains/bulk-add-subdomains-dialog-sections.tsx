"use client"

import React from "react"
import { AlertCircle, CheckCircle, ChevronUp, Info, Plus, semanticIcons } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { getStatusToneSurfaceClass, getStatusToneTextClass } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import {
  LineNumberedTextarea,
  lineNumberedTextareaResponsiveViewportClassName,
  type LineNumberedTextareaHighlight,
} from "@/components/common/line-numbered-textarea"
import type {
  BulkAddSubdomainLineIssue,
  BulkAddSubdomainLineIssueType,
} from "@/components/subdomains/bulk-add-subdomains-dialog-state"
import type { SubdomainValidationErrorCode } from "@/lib/subdomain-validator"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface BulkAddSubdomainsHeaderProps {
  title: string
  description: string
  targetName?: string
  t: TranslationFn
}

export function BulkAddSubdomainsHeader({
  title,
  description,
  targetName,
  t,
}: BulkAddSubdomainsHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle className="flex items-center space-x-2">
        <semanticIcons.concept.subdomain className="h-5 w-5" />
        <span>{title}</span>
      </DialogTitle>
      <DialogDescription>
        {description}
        {targetName && (
          <span className="block mt-1">
            {t("belongsTo")} <code className="bg-muted px-1 rounded">{targetName}</code>
          </span>
        )}
      </DialogDescription>
    </DialogHeader>
  )
}

interface BulkAddSubdomainsInputProps {
  t: TranslationFn
  inputText: string
  placeholder: string
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
    blockingIssueCount: number
    advisoryIssueCount: number
    invalidItems: Array<{ lineNumber: number; subdomain: string; error?: string; errorCode?: SubdomainValidationErrorCode }>
    duplicateItems: Array<{ lineNumber: number; subdomain: string; duplicateOfLine: number }>
    lineIssues: BulkAddSubdomainLineIssue[]
  } | null
}

function getSubdomainValidationErrorMessage(t: TranslationFn, errorCode?: SubdomainValidationErrorCode): string {
  return t(`errors.${errorCode || "unknown"}`)
}

function getIssueBadgeLabel(t: TranslationFn, type: BulkAddSubdomainLineIssueType): string {
  if (type === "duplicate") return t("duplicateBadge")
  return t("invalidBadge")
}

function getIssueMessage(t: TranslationFn, issue: BulkAddSubdomainLineIssue): string {
  if (issue.type === "duplicate") {
    return t("duplicateIssue", { line: issue.lineNumber })
  }

  return t("invalidIssue", {
    line: issue.lineNumber,
    error: getSubdomainValidationErrorMessage(t, issue.errorCode),
  })
}

export function BulkAddSubdomainsInput({
  t,
  inputText,
  placeholder,
  lineNumbersRef,
  textareaRef,
  onInputChange,
  onScroll,
  isPending,
  lineCount,
  validationResult,
}: BulkAddSubdomainsInputProps) {
  const [detailsOpen, setDetailsOpen] = React.useState(true)
  const lineHighlights = React.useMemo<LineNumberedTextareaHighlight[]>(() => (
    validationResult?.lineIssues.map((issue) => ({
      lineNumber: issue.lineNumber,
      tone: issue.tone,
      label: getIssueBadgeLabel(t, issue.type),
    })) ?? []
  ), [t, validationResult])

  return (
    <div className="gap-3 grid">
      <Label htmlFor="subdomains">
        {t("label")} <span className="text-destructive">*</span>
      </Label>
      <LineNumberedTextarea
        lineCount={lineCount}
        lineNumbersRef={lineNumbersRef}
        textareaRef={textareaRef}
        id="subdomains"
        name="subdomains"
        autoComplete="off"
        value={inputText}
        onChange={(event) => onInputChange(event.target.value)}
        onScroll={onScroll}
        viewportClassName={lineNumberedTextareaResponsiveViewportClassName}
        placeholder={placeholder}
        disabled={isPending}
        lineHighlights={lineHighlights}
      />

      <div className={cn("flex gap-2", textRole.helperText)}>
        <Info className="mt-0.5 h-4 w-4 shrink-0" />
        <div className="grid gap-1">
          <span>{t("helper")}</span>
          <span>{t("example")}</span>
        </div>
      </div>

      {!validationResult && (
        <div className={cn("flex items-center gap-2 rounded-md border bg-muted/20 px-3 py-2", textRole.bodyStrong)}>
          <Info className={cn("h-4 w-4", getStatusToneTextClass("muted"))} />
          <span>{t("emptySummary")}</span>
        </div>
      )}

      {validationResult && validationResult.blockingIssueCount === 0 && (
        <div className={cn("flex items-center gap-2 rounded-md px-3 py-2", getStatusToneSurfaceClass("success"), textRole.bodyStrong)}>
          <CheckCircle className={cn("h-4 w-4", getStatusToneTextClass("success"))} />
          <span>{t("validSummary", { count: validationResult.validCount })}</span>
          {validationResult.advisoryIssueCount > 0 && (
            <span className={cn("ml-1", getStatusToneTextClass("warning"), textRole.helperText)}>
              {t("advisorySummary", { count: validationResult.advisoryIssueCount })}
            </span>
          )}
        </div>
      )}

      {validationResult && validationResult.blockingIssueCount > 0 && (
        <div className={cn("overflow-hidden rounded-md", getStatusToneSurfaceClass("error"))}>
          <div className="flex items-start justify-between gap-3 border-b border-error/20 bg-error/10 px-3 py-2">
            <div className="grid gap-1">
              <div className={cn("flex items-center gap-2", getStatusToneTextClass("error"), textRole.bodyStrong)}>
                <AlertCircle className="h-4 w-4" />
                <span>{t("blockingSummary", { count: validationResult.blockingIssueCount })}</span>
              </div>
              {validationResult.advisoryIssueCount > 0 && (
                <div className={cn("flex items-center gap-2", getStatusToneTextClass("warning"), textRole.helperText)}>
                  <AlertCircle className="h-3.5 w-3.5" />
                  <span>{t("advisorySummary", { count: validationResult.advisoryIssueCount })}</span>
                </div>
              )}
            </div>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className={cn("h-auto gap-1 px-1 py-0", getStatusToneTextClass("info"), textRole.helperText)}
              onClick={() => setDetailsOpen((open) => !open)}
            >
              {detailsOpen ? t("collapseDetails") : t("expandDetails")}
              <ChevronUp className={cn("h-3.5 w-3.5 transition-transform", !detailsOpen && "rotate-180")} />
            </Button>
          </div>
          {detailsOpen && (
            <ul className="grid gap-2 px-4 py-3">
              {validationResult.lineIssues.map((issue) => (
                <li key={`${issue.type}-${issue.lineNumber}`} className={cn("flex items-start gap-2", textRole.helperText)}>
                  <span
                    className={cn(
                      "mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full",
                      issue.tone === "error" ? "bg-error" : "bg-warning"
                    )}
                  />
                  <span className={issue.tone === "error" ? "text-foreground" : getStatusToneTextClass("warning")}>
                    {getIssueMessage(t, issue)}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}

interface BulkAddSubdomainsFooterProps {
  t: TranslationFn
  isPending: boolean
  isFormValid: boolean
}

export function BulkAddSubdomainsFooter({
  t,
  isPending,
  isFormValid,
}: BulkAddSubdomainsFooterProps) {
  return (
    <DialogFooter>
      <Button
        type="submit"
        disabled={isPending || !isFormValid}
        loading={isPending}
        loadingLabel={t("creating")}
      >
        <>
          <Plus className="h-4 w-4" />
          {t("bulkAdd")}
        </>
      </Button>
    </DialogFooter>
  )
}
