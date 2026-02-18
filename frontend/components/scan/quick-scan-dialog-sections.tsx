"use client"

import React from "react"
import { AlertCircle, ChevronLeft, ChevronRight, Zap } from "@/components/icons"
import { Button } from "@/components/ui/button"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Textarea } from "@/components/ui/textarea"
import { LoadingSpinner } from "@/components/loading-spinner"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface QuickScanTriggerProps {
  t: TranslationFn
  trigger?: React.ReactElement
}

export function QuickScanTrigger({ t, trigger }: QuickScanTriggerProps) {
  if (trigger) return trigger

  return (
    <Button
      variant="outline"
      size="sm"
      className="
          relative group border-0
          bg-secondary/50
          px-4 overflow-hidden
        "
    >
      <div className="absolute inset-0 border border-highlight/0 group-hover:border-highlight/100 transition-colors duration-300 scale-95 group-hover:scale-100" />
      <div className="absolute top-0 left-0 w-1 h-1 bg-highlight opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
      <div className="absolute top-0 right-0 w-1 h-1 bg-highlight opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
      <div className="absolute bottom-0 left-0 w-1 h-1 bg-highlight opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
      <div className="absolute bottom-0 right-0 w-1 h-1 bg-highlight opacity-0 group-hover:opacity-100 transition-opacity duration-300" />

      <span className="mr-2 text-xs font-mono text-highlight opacity-0 group-hover:opacity-100 absolute left-2 transition-[opacity,transform] duration-300 -translate-x-2 group-hover:translate-x-0">{">"}</span>
      <div className="flex items-center transition-transform duration-300 group-hover:translate-x-2">
        <span className="font-medium">{t("title")}</span>
      </div>
    </Button>
  )
}

interface QuickScanHeaderProps {
  t: TranslationFn
  step: number
  totalSteps: number
}

export function QuickScanHeader({ t, step, totalSteps }: QuickScanHeaderProps) {
  return (
    <DialogHeader className="px-6 pt-6 pb-4">
      <div className="flex items-center justify-between">
        <div>
          <DialogTitle className="flex items-center gap-2">
            <Zap className="h-5 w-5 text-primary" />
            {t("title")}
          </DialogTitle>
          <DialogDescription className="mt-1">
            {t("description")}
          </DialogDescription>
        </div>
        <div className="text-sm text-muted-foreground mr-8">
          {t("stepIndicator", { current: step, total: totalSteps })}
        </div>
      </div>
    </DialogHeader>
  )
}

interface QuickScanTargetStepProps {
  t: TranslationFn
  targetInput: string
  onTargetChange: (value: string) => void
  lineNumbersRef: React.RefObject<HTMLDivElement | null>
  onScroll: (event: React.UIEvent<HTMLTextAreaElement>) => void
  invalidInputs: Array<{ lineNumber: number; error?: string }>
  hasErrors: boolean
}

export function QuickScanTargetStep({
  t,
  targetInput,
  onTargetChange,
  lineNumbersRef,
  onScroll,
  invalidInputs,
  hasErrors,
}: QuickScanTargetStepProps) {
  return (
    <div className="flex flex-col h-full">
      <div className="px-4 py-3 border-b bg-muted/30 shrink-0">
        <h3 className="text-sm leading-6">
          <span className="font-medium">{t("scanTargets")}</span>
          <span className="text-muted-foreground">：{t("supportedFormats")}</span>
        </h3>
      </div>
      <div className="flex-1 flex flex-col overflow-hidden">
        <div className="flex-1 flex overflow-hidden">
          <div className="flex-shrink-0 w-10 bg-muted/30">
            <div
              ref={lineNumbersRef}
              className="py-3 px-2 text-right font-mono text-xs text-muted-foreground leading-[1.5] h-full overflow-y-auto scrollbar-hide"
            >
              {Array.from({ length: Math.max(targetInput.split("\n").length, 20) }, (_, i) => (
                <div key={i + 1} className="h-[21px]">{i + 1}</div>
              ))}
            </div>
          </div>
          <div className="flex-1 overflow-hidden">
            <Textarea
              name="scanTargets"
              autoComplete="off"
              value={targetInput}
              onChange={(event) => onTargetChange(event.target.value)}
              onScroll={onScroll}
              placeholder={t("targetPlaceholder")}
              className="font-mono h-full overflow-y-auto resize-none border-0 rounded-none focus-visible:ring-0 focus-visible:ring-offset-0 text-sm py-3 px-3"
              style={{ lineHeight: "21px" }}
            />
          </div>
        </div>
        {hasErrors && (
          <div className="px-3 py-2 border-t bg-destructive/5 max-h-[60px] overflow-y-auto">
            {invalidInputs.slice(0, 2).map((result) => (
              <div key={result.lineNumber} className="flex items-center gap-1 text-xs text-destructive">
                <AlertCircle className="h-3 w-3 shrink-0" />
                <span>{t("lineError", { lineNumber: result.lineNumber, error: result.error || "" })}</span>
              </div>
            ))}
            {invalidInputs.length > 2 && (
              <div className="text-xs text-muted-foreground">
                {t("moreErrors", { count: invalidInputs.length - 2 })}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

interface QuickScanFooterProps {
  t: TranslationFn
  step: number
  validCount: number
  invalidCount: number
  selectedEngineCount: number
  isSubmitting: boolean
  canProceedToStep2: boolean
  canProceedToStep3: boolean
  canSubmit: boolean
  onBack: () => void
  onNext: () => void
  onSubmit: () => void
}

export function QuickScanFooter({
  t,
  step,
  validCount,
  invalidCount,
  selectedEngineCount,
  isSubmitting,
  canProceedToStep2,
  canProceedToStep3,
  canSubmit,
  onBack,
  onNext,
  onSubmit,
}: QuickScanFooterProps) {
  return (
    <div className="px-4 py-4 border-t !flex !items-center !justify-between">
      <div className="text-sm">
        {step === 1 && validCount > 0 && (
          <span className="text-primary">{t("validTargets", { count: validCount })}</span>
        )}
        {step === 1 && invalidCount > 0 && (
          <span className="text-destructive ml-2">{t("invalidTargets", { count: invalidCount })}</span>
        )}
        {step === 2 && selectedEngineCount > 0 && (
          <span className="text-primary">{t("selectedCount", { count: selectedEngineCount })}</span>
        )}
      </div>
      <div className="flex gap-2">
        {step > 1 && (
          <Button variant="outline" onClick={onBack} disabled={isSubmitting}>
            <ChevronLeft className="h-4 w-4 mr-1" />
            {t("back")}
          </Button>
        )}
        {step < 3 ? (
          <Button
            onClick={onNext}
            disabled={step === 1 ? !canProceedToStep2 : !canProceedToStep3}
          >
            {t("next")}
            <ChevronRight className="h-4 w-4 ml-1" />
          </Button>
        ) : (
          <Button onClick={onSubmit} disabled={!canSubmit || isSubmitting}>
            {isSubmitting ? (
              <>
                <LoadingSpinner />
                {t("creating")}
              </>
            ) : (
              <>
                <Zap className="h-4 w-4" />
                {t("startScan")}
              </>
            )}
          </Button>
        )}
      </div>
    </div>
  )
}

interface QuickScanOverwriteDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCancel: () => void
  onConfirm: () => void
  t: TranslationFn
}

export function QuickScanOverwriteDialog({
  open,
  onOpenChange,
  onCancel,
  onConfirm,
  t,
}: QuickScanOverwriteDialogProps) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("overwriteConfirm.title")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t("overwriteConfirm.description")}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={onCancel}>
            {t("overwriteConfirm.cancel")}
          </AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm}>
            {t("overwriteConfirm.confirm")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
