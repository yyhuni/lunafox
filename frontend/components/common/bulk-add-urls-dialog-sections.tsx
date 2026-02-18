"use client"

import React from "react"
import { Link, Plus } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { LoadingSpinner } from "@/components/loading-spinner"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface BulkAddUrlsDialogHeaderProps {
  title: string
  description: string
}

export function BulkAddUrlsDialogHeader({ title, description }: BulkAddUrlsDialogHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle className="flex items-center space-x-2">
        <Link className="h-5 w-5" />
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
    firstError?: { index: number; url: string; error: string }
    firstMismatch?: { index: number; url: string }
  } | null
  targetName?: string
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
  targetName,
}: BulkAddUrlsInputProps) {
  return (
    <div className="grid gap-2">
      <Label htmlFor="urls">
        {tUrl("label")} <span className="text-destructive">*</span>
      </Label>
      <div className="flex border rounded-md overflow-hidden h-[220px]">
        <div className="flex-shrink-0 w-12 border-r bg-muted/50">
          <div
            ref={lineNumbersRef}
            className="py-3 px-2 text-right font-mono text-xs text-muted-foreground leading-[1.4] h-full overflow-y-auto scrollbar-hide"
          >
            {Array.from({ length: lineCount }, (_, i) => (
              <div key={i + 1} className="h-[20px]">
                {i + 1}
              </div>
            ))}
          </div>
        </div>
        <div className="flex-1 overflow-hidden">
          <Textarea
            ref={textareaRef}
            id="urls"
            name="urls"
            autoComplete="off"
            value={inputText}
            onChange={(event) => onInputChange(event.target.value)}
            onScroll={onScroll}
            placeholder={placeholder}
            disabled={isPending}
            className="font-mono h-full overflow-y-auto resize-none border-0 focus-visible:ring-0 focus-visible:ring-offset-0 leading-[1.4] text-sm py-3"
            style={{ lineHeight: "20px" }}
          />
        </div>
      </div>

      {validationResult && (
        <div className="text-xs space-y-1">
          <div className="text-muted-foreground">
            {tUrl("valid", { count: validationResult.validCount })}
            {validationResult.duplicateCount > 0 && (
              <span className="text-yellow-600 ml-2">
                {tUrl("duplicate", { count: validationResult.duplicateCount })}
              </span>
            )}
            {validationResult.invalidCount > 0 && (
              <span className="text-destructive ml-2">
                {tUrl("invalid", { count: validationResult.invalidCount })}
              </span>
            )}
            {validationResult.mismatchedCount > 0 && (
              <span className="text-destructive ml-2">
                {tUrl("mismatched", { count: validationResult.mismatchedCount })}
              </span>
            )}
          </div>
          {validationResult.firstError && (
            <div className="text-destructive">
              {tUrl("lineError", {
                line: validationResult.firstError.index + 1,
                value: validationResult.firstError.url.length > 50
                  ? `${validationResult.firstError.url.substring(0, 50)}...`
                  : validationResult.firstError.url,
                error: validationResult.firstError.error,
              })}
            </div>
          )}
          {validationResult.firstMismatch && !validationResult.firstError && (
            <div className="text-destructive">
              {tUrl("mismatchError", {
                line: validationResult.firstMismatch.index + 1,
                value: validationResult.firstMismatch.url.length > 50
                  ? `${validationResult.firstMismatch.url.substring(0, 50)}...`
                  : validationResult.firstMismatch.url,
                target: targetName || "",
              })}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

interface BulkAddUrlsFooterProps {
  tBulkAdd: TranslationFn
  tUrl: TranslationFn
  onCancel: () => void
  isPending: boolean
  isFormValid: boolean
}

export function BulkAddUrlsFooter({
  tBulkAdd,
  tUrl,
  onCancel,
  isPending,
  isFormValid,
}: BulkAddUrlsFooterProps) {
  return (
    <DialogFooter>
      <Button
        type="button"
        variant="outline"
        onClick={onCancel}
        disabled={isPending}
      >
        {tBulkAdd("cancel")}
      </Button>
      <Button
        type="submit"
        disabled={isPending || !isFormValid}
      >
        {isPending ? (
          <>
            <LoadingSpinner />
            {tUrl("creating")}
          </>
        ) : (
          <>
            <Plus className="h-4 w-4" />
            {tBulkAdd("bulkAdd")}
          </>
        )}
      </Button>
    </DialogFooter>
  )
}
