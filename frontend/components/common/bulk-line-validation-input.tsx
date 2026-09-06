"use client"

import React from "react"
import { AlertCircle, CheckCircle, ChevronUp, Info } from "@/components/icons"

import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import {
  LineNumberedTextarea,
  countLineNumberedTextareaLines,
  lineNumberedTextareaFillViewportClassName,
  lineNumberedTextareaResponsiveViewportClassName,
  type LineNumberedTextareaHighlight,
  type LineNumberedTextareaHighlightTone,
} from "@/components/common/line-numbered-textarea"
import { getStatusToneSurfaceClass, getStatusToneTextClass } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

export interface BulkLineValidationIssue {
  id: string
  lineNumber: number
  tone: LineNumberedTextareaHighlightTone
  badgeLabel: string
  message: string
}

export interface BulkLineValidationResult {
  validCount: number
  blockingIssueCount: number
  advisoryIssueCount: number
  lineIssues: BulkLineValidationIssue[]
}

interface BulkLineValidationInputProps {
  id: string
  name: string
  label: string
  labelClassName?: string
  required?: boolean
  placeholder: string
  value: string
  lineCount: number
  lineNumbersRef: React.RefObject<HTMLDivElement | null>
  textareaRef: React.RefObject<HTMLTextAreaElement | null>
  onValueChange: (value: string) => void
  onScroll: (event: React.UIEvent<HTMLTextAreaElement>) => void
  onTextareaRef?: (element: HTMLTextAreaElement | null) => void
  disabled?: boolean
  helper: string
  example: string
  emptySummary: string
  validSummary: string
  blockingSummary: string
  advisorySummary?: string
  collapseDetails: string
  expandDetails: string
  validationResult: BulkLineValidationResult | null
  showEmptySummary?: boolean
  showSuccessSummary?: boolean
  viewportClassName?: string
  fillHeight?: boolean
}

export function BulkLineValidationInput({
  id,
  name,
  label,
  labelClassName,
  required,
  placeholder,
  value,
  lineCount,
  lineNumbersRef,
  textareaRef,
  onValueChange,
  onScroll,
  onTextareaRef,
  disabled,
  helper,
  example,
  emptySummary,
  validSummary,
  blockingSummary,
  advisorySummary,
  collapseDetails,
  expandDetails,
  validationResult,
  showEmptySummary = true,
  showSuccessSummary = true,
  viewportClassName = lineNumberedTextareaResponsiveViewportClassName,
  fillHeight = false,
}: BulkLineValidationInputProps) {
  const [detailsOpen, setDetailsOpen] = React.useState(false)
  const effectiveLineCount = Math.max(lineCount, countLineNumberedTextareaLines(value))
  const lineHighlights = React.useMemo<LineNumberedTextareaHighlight[]>(() => (
    validationResult?.lineIssues.map((issue) => ({
      lineNumber: issue.lineNumber,
      tone: issue.tone,
      label: issue.badgeLabel,
    })) ?? []
  ), [validationResult])

  return (
    <div className={cn("gap-3", fillHeight ? "flex h-full min-h-0 flex-col" : "grid")}>
      <Label htmlFor={id} className={labelClassName}>
        {label} {required && <span className="text-destructive">*</span>}
      </Label>
      <LineNumberedTextarea
        lineCount={effectiveLineCount}
        lineNumbersRef={lineNumbersRef}
        textareaRef={textareaRef}
        viewportClassName={cn(viewportClassName, fillHeight && lineNumberedTextareaFillViewportClassName)}
        id={id}
        name={name}
        autoComplete="off"
        value={value}
        onChange={(event) => onValueChange(event.target.value)}
        onScroll={onScroll}
        onTextareaRef={onTextareaRef}
        placeholder={placeholder}
        disabled={disabled}
        lineHighlights={lineHighlights}
      />

      {showEmptySummary && !validationResult && (
        <div className="flex items-start gap-2 rounded-md border bg-muted/20 px-3 py-2">
          <Info className={cn("mt-0.5 h-4 w-4 shrink-0", getStatusToneTextClass("muted"))} />
          <div className="grid gap-1">
            <span className={textRole.bodyStrong}>{emptySummary}</span>
            <span className={textRole.helperText}>{helper}</span>
            <span className={textRole.helperText}>{example}</span>
          </div>
        </div>
      )}

      {showSuccessSummary && validationResult && validationResult.blockingIssueCount === 0 && (
        <div className={cn("flex items-center gap-2 rounded-md px-3 py-2", getStatusToneSurfaceClass("success"), textRole.bodyStrong)}>
          <CheckCircle className={cn("h-4 w-4", getStatusToneTextClass("success"))} />
          <span>{validSummary}</span>
          {validationResult.advisoryIssueCount > 0 && advisorySummary && (
            <span className={cn("ml-1", getStatusToneTextClass("warning"), textRole.helperText)}>
              {advisorySummary}
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
                <span>{blockingSummary}</span>
              </div>
              {validationResult.advisoryIssueCount > 0 && advisorySummary && (
                <div className={cn("flex items-center gap-2", getStatusToneTextClass("warning"), textRole.helperText)}>
                  <AlertCircle className="h-3.5 w-3.5" />
                  <span>{advisorySummary}</span>
                </div>
              )}
            </div>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className={cn(
                "h-auto gap-1 px-1 py-0 hover:bg-transparent dark:hover:bg-transparent",
                getStatusToneTextClass("info"),
                textRole.helperText
              )}
              onClick={() => setDetailsOpen((open) => !open)}
            >
              {detailsOpen ? collapseDetails : expandDetails}
              <ChevronUp className={cn("h-3.5 w-3.5 transition-transform", !detailsOpen && "rotate-180")} />
            </Button>
          </div>
          {detailsOpen && (
            <ul className="grid gap-2 px-4 py-3">
              {validationResult.lineIssues.map((issue) => (
                <li key={issue.id} className={cn("flex items-start gap-2", textRole.helperText)}>
                  <span
                    className={cn(
                      "mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full",
                      issue.tone === "error" ? "bg-error" : "bg-warning"
                    )}
                  />
                  <span className={issue.tone === "error" ? "text-foreground" : getStatusToneTextClass("warning")}>
                    {issue.message}
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
