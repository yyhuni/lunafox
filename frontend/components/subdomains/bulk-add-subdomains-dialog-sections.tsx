"use client"

import React from "react"
import { Globe, Plus } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { LoadingSpinner } from "@/components/loading-spinner"

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
        <Globe className="h-5 w-5" />
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
    firstError?: { index: number; subdomain: string; error: string }
  } | null
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
  return (
    <div className="grid gap-2">
      <Label htmlFor="subdomains">
        {t("label")} <span className="text-destructive">*</span>
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
            id="subdomains"
            name="subdomains"
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
            {t("valid", { count: validationResult.validCount })}
            {validationResult.duplicateCount > 0 && (
              <span className="text-yellow-600 ml-2">
                {t("duplicate", { count: validationResult.duplicateCount })}
              </span>
            )}
            {validationResult.invalidCount > 0 && (
              <span className="text-destructive ml-2">
                {t("invalid", { count: validationResult.invalidCount })}
              </span>
            )}
          </div>
          {validationResult.firstError && (
            <div className="text-destructive">
              {t("lineError", {
                line: validationResult.firstError.index + 1,
                value: validationResult.firstError.subdomain,
                error: validationResult.firstError.error,
              })}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

interface BulkAddSubdomainsFooterProps {
  tCommon: TranslationFn
  t: TranslationFn
  onCancel: () => void
  isPending: boolean
  isFormValid: boolean
}

export function BulkAddSubdomainsFooter({
  tCommon,
  t,
  onCancel,
  isPending,
  isFormValid,
}: BulkAddSubdomainsFooterProps) {
  return (
    <DialogFooter>
      <Button
        type="button"
        variant="outline"
        onClick={onCancel}
        disabled={isPending}
      >
        {tCommon("cancel")}
      </Button>
      <Button
        type="submit"
        disabled={isPending || !isFormValid}
      >
        {isPending ? (
          <>
            <LoadingSpinner />
            {t("creating")}
          </>
        ) : (
          <>
            <Plus className="h-4 w-4" />
            {t("bulkAdd")}
          </>
        )}
      </Button>
    </DialogFooter>
  )
}
