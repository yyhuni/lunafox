"use client"

import React from "react"
import { ChevronLeft, ChevronRight, semanticIcons, Zap } from "@/components/icons"
import { Button } from "@/components/ui/button"
import {
  DrawerClose,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer"
import { AddTargetInputSection } from "@/components/target/add-target-dialog-sections"
import { EdgePanelHeader } from "@/components/shared/edge-panel-header"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface QuickScanHeaderProps {
  t: TranslationFn
  step: number
  totalSteps: number
  closeLabel: string
  isSubmitting: boolean
  stepHeader: React.ReactNode
}

export function QuickScanHeader({
  t,
  step,
  totalSteps,
  closeLabel,
  isSubmitting,
  stepHeader,
}: QuickScanHeaderProps) {
  return (
    <DrawerHeader className="shrink-0 bg-card px-5 pt-4 text-left sm:px-6">
      <EdgePanelHeader
        variant="workbench"
        leading={<Zap className="size-7" />}
        title={<DrawerTitle className={textRole.sectionTitle}>{t("title")}</DrawerTitle>}
        description={(
          <DrawerDescription className={cn("max-w-2xl", textRole.helperText)}>
            {t("description")}
          </DrawerDescription>
        )}
        actions={(
          <>
            <span className={cn("hidden sm:inline", textRole.helperText)}>
              {t("stepIndicator", { current: step, total: totalSteps })}
            </span>
            <DrawerClose
              render={(
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  aria-label={closeLabel}
                  disabled={isSubmitting}
                />
              )}
            >
              <semanticIcons.action.cancel />
            </DrawerClose>
          </>
        )}
        progress={stepHeader}
      />
    </DrawerHeader>
  )
}

interface QuickScanTargetStepProps {
  tTargetDialog: TranslationFn
  targetInput: string
  validCount: number
  onTargetChange: (value: string) => void
  lineNumbersRef: React.RefObject<HTMLDivElement | null>
  textareaRef: React.RefObject<HTMLTextAreaElement | null>
  onScroll: (event: React.UIEvent<HTMLTextAreaElement>) => void
  invalidInputs: Array<{ lineNumber: number; originalInput: string; error?: string }>
  isSubmitting: boolean
}

export function QuickScanTargetStep({
  tTargetDialog,
  targetInput,
  validCount,
  onTargetChange,
  lineNumbersRef,
  textareaRef,
  onScroll,
  invalidInputs,
  isSubmitting,
}: QuickScanTargetStepProps) {
  return (
    <AddTargetInputSection
      t={tTargetDialog}
      formTargets={targetInput}
      targetCount={validCount}
      invalidTargets={invalidInputs.map((input, index) => ({
        index,
        lineNumber: input.lineNumber,
        originalTarget: input.originalInput,
        error: input.error || tTargetDialog("invalidFormat"),
      }))}
      lineNumbersRef={lineNumbersRef}
      textareaRef={textareaRef}
      onInputChange={onTargetChange}
      onScroll={onScroll}
      isPending={isSubmitting}
    />
  )
}

interface QuickScanFooterProps {
  t: TranslationFn
  step: number
  validCount: number
  invalidCount: number
  selectedScanWorkflowCount: number
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
  selectedScanWorkflowCount,
  isSubmitting,
  canProceedToStep2,
  canProceedToStep3,
  canSubmit,
  onBack,
  onNext,
  onSubmit,
}: QuickScanFooterProps) {
  return (
    <div className="flex shrink-0 flex-col gap-3 md:flex-row md:items-center md:justify-between">
      {/* Keep the navigation actions anchored while later steps show their two-line status. */}
      <div className="flex min-h-10 items-center text-sm">
        {step === 1 && validCount > 0 && (
          <span className="text-primary">{t("validTargets", { count: validCount })}</span>
        )}
        {step === 1 && invalidCount > 0 && (
          <span className="ml-2 text-destructive">{t("invalidTargets", { count: invalidCount })}</span>
        )}
        {step === 2 && selectedScanWorkflowCount > 0 && (
          <span className="text-primary">{t("selectedCount", { count: selectedScanWorkflowCount })}</span>
        )}
      </div>
      <div className="flex items-center justify-between gap-3 md:justify-end">
        <div className="flex gap-2">
          {step > 1 && (
            <Button type="button" variant="outline" size="sm" onClick={onBack} disabled={isSubmitting}>
              <ChevronLeft className="size-4" />
              {t("back")}
            </Button>
          )}
        </div>
        {step < 3 ? (
          <Button
            type="button"
            size="sm"
            onClick={onNext}
            disabled={step === 1 ? !canProceedToStep2 : !canProceedToStep3}
          >
            {t("next")}
            <ChevronRight className="size-4" />
          </Button>
        ) : (
          <Button
            type="button"
            size="sm"
            onClick={onSubmit}
            disabled={!canSubmit || isSubmitting}
            loading={isSubmitting}
            loadingLabel={t("creating")}
          >
            <>
              <Zap className="h-4 w-4" />
              {t("startScan")}
            </>
          </Button>
        )}
      </div>
    </div>
  )
}
