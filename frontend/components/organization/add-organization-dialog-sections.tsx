"use client"

import React from "react"
import type { Control, FieldValues, Path } from "react-hook-form"
import { ChevronRight, semanticIcons } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { BulkLineValidationInput, type BulkLineValidationIssue } from "@/components/common/bulk-line-validation-input"
import { countLineNumberedTextareaLines } from "@/components/common/line-numbered-textarea"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

const AddIcon = semanticIcons.action.add

interface AddOrganizationNameFieldProps<TFieldValues extends FieldValues> {
  t: TranslationFn
  formControl: Control<TFieldValues>
  isSubmitting: boolean
  name: Path<TFieldValues>
}

export function AddOrganizationNameField<TFieldValues extends FieldValues>({
  t,
  formControl,
  isSubmitting,
  name,
}: AddOrganizationNameFieldProps<TFieldValues>) {
  return (
    <FormField
      control={formControl}
      name={name}
      render={({ field }) => (
        <FormItem>
          <FormLabel>
            {t("orgName")} <span className="text-destructive">*</span>
          </FormLabel>
          <FormControl>
            <Input
              placeholder={t("orgNamePlaceholder")}
              disabled={isSubmitting}
              maxLength={50}
              autoComplete="off"
              {...field}
            />
          </FormControl>
          <FormDescription>
            {t("characters", { count: field.value.length, max: 50 })}
          </FormDescription>
          <FormMessage />
        </FormItem>
      )}
    />
  )
}

interface AddOrganizationDescriptionFieldProps<TFieldValues extends FieldValues> {
  t: TranslationFn
  formControl: Control<TFieldValues>
  isSubmitting: boolean
  name: Path<TFieldValues>
}

export function AddOrganizationDescriptionField<TFieldValues extends FieldValues>({
  t,
  formControl,
  isSubmitting,
  name,
}: AddOrganizationDescriptionFieldProps<TFieldValues>) {
  return (
    <FormField
      control={formControl}
      name={name}
      render={({ field }) => (
        <FormItem>
          <FormLabel>{t("orgDescOptional")}</FormLabel>
          <FormControl>
            <Textarea
              placeholder={t("orgDescPlaceholder")}
              disabled={isSubmitting}
              rows={3}
              maxLength={200}
              autoComplete="off"
              {...field}
            />
          </FormControl>
          <FormDescription>
            {t("characters", { count: (field.value || "").length, max: 200 })}
          </FormDescription>
          <FormMessage />
        </FormItem>
      )}
    />
  )
}

interface AddOrganizationTargetsFieldProps<TFieldValues extends FieldValues> {
  t: TranslationFn
  formControl: Control<TFieldValues>
  isSubmitting: boolean
  lineNumbersRef: React.RefObject<HTMLDivElement | null>
  textareaRef: React.RefObject<HTMLTextAreaElement | null>
  onScroll: (event: React.UIEvent<HTMLTextAreaElement>) => void
  isTargetsExpanded: boolean
  onToggleTargetsExpanded: () => void
  targetValidation: {
    count: number
    invalid: Array<{ index: number; lineNumber: number; originalTarget: string; error: string }>
  }
  name: Path<TFieldValues>
}

export function AddOrganizationTargetsField<TFieldValues extends FieldValues>({
  t,
  formControl,
  isSubmitting,
  lineNumbersRef,
  textareaRef,
  onScroll,
  isTargetsExpanded,
  onToggleTargetsExpanded,
  targetValidation,
  name,
}: AddOrganizationTargetsFieldProps<TFieldValues>) {
  const targetsPanelId = React.useId()

  return (
    <FormField
      control={formControl}
      name={name}
      render={({ field }) => (
        <FormItem className="mt-2 gap-3 border-t border-border/60 pt-4">
          <div className="flex items-start justify-between gap-3">
            <Button
              type="button"
              variant="ghost"
              className="group h-auto min-w-0 flex-1 justify-start gap-2 px-0 py-0 text-left hover:bg-transparent hover:text-foreground dark:hover:bg-transparent dark:hover:text-foreground"
              aria-expanded={isTargetsExpanded}
              aria-controls={targetsPanelId}
              data-panel-open={isTargetsExpanded ? "" : undefined}
              onClick={onToggleTargetsExpanded}
              disabled={isSubmitting}
            >
              <span className="flex min-w-0 items-center gap-2">
                <span className={cn(textRole.bodyStrong, "text-muted-foreground transition-colors group-hover:text-foreground")}>
                  {t("addTargets")}
                </span>
                <ChevronRight className="size-4 shrink-0 text-muted-foreground transition-[color,transform] duration-200 motion-reduce:transition-none group-hover:text-foreground group-data-[panel-open]:rotate-90" />
              </span>
            </Button>
          </div>

          {isTargetsExpanded && (
            <div id={targetsPanelId} className="grid gap-2">
              {(() => {
                const fieldValue = typeof field.value === "string" ? field.value : ""
                const lineIssues: BulkLineValidationIssue[] = targetValidation.invalid.map((target) => ({
                  id: `invalid-${target.index}`,
                  lineNumber: target.lineNumber,
                  tone: "error",
                  badgeLabel: t("targetInvalidBadge"),
                  message: t("targetInvalidIssue", {
                    line: target.lineNumber,
                    target: target.originalTarget,
                    error: target.error,
                  }),
                }))
                const validationResult = fieldValue.trim().length > 0
                  ? {
                      validCount: targetValidation.count,
                      blockingIssueCount: targetValidation.invalid.length,
                      advisoryIssueCount: 0,
                      lineIssues,
                    }
                  : null

                return (
                  <BulkLineValidationInput
                    id="organization-targets"
                    name={field.name}
                    label={t("targetList")}
                    placeholder={t("targetsPlaceholder")}
                    value={fieldValue}
                    lineCount={Math.max(countLineNumberedTextareaLines(fieldValue), 8)}
                    lineNumbersRef={lineNumbersRef}
                    textareaRef={textareaRef}
                    onTextareaRef={field.ref}
                    onValueChange={field.onChange}
                    onScroll={onScroll}
                    disabled={isSubmitting}
                    helper={t("targetHelp")}
                    example={t("targetExample")}
                    emptySummary={t("targetEmptySummary")}
                    validSummary={t("targetValidSummary", { count: targetValidation.count })}
                    blockingSummary={t("targetBlockingSummary", { count: targetValidation.invalid.length })}
                    collapseDetails={t("collapseDetails")}
                    expandDetails={t("expandDetails")}
                    validationResult={validationResult}
                  />
                )
              })()}
              <FormMessage />
            </div>
          )}
        </FormItem>
      )}
    />
  )
}

interface AddOrganizationFooterProps {
  t: TranslationFn
  isSubmitting: boolean
  isFormValid: boolean
  createPending: boolean
}

export function AddOrganizationFooter({
  t,
  isSubmitting,
  isFormValid,
  createPending,
}: AddOrganizationFooterProps) {
  return (
    <div className="flex justify-end gap-2">
      <Button
        type="submit"
        disabled={isSubmitting || !isFormValid}
        loading={isSubmitting}
        loadingLabel={createPending ? t("creating") : t("creatingTargets")}
      >
        <>
          <AddIcon />
          {t("create")}
        </>
      </Button>
    </div>
  )
}
