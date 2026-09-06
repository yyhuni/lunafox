"use client"

import React from "react"
import type { Control, FieldValues, Path } from "react-hook-form"
import { semanticIcons } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { BulkLineValidationInput, type BulkLineValidationIssue } from "@/components/common/bulk-line-validation-input"
import {
  countLineNumberedTextareaLines,
  lineNumberedTextareaResponsiveViewportClassName,
} from "@/components/common/line-numbered-textarea"
import {
  FormField,
  FormItem,
  FormMessage,
} from "@/components/ui/form"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

const AddIcon = semanticIcons.action.add
const OrganizationIcon = semanticIcons.concept.organization

interface LinkTargetInputSectionProps<TFieldValues extends FieldValues> {
  t: TranslationFn
  formControl: Control<TFieldValues>
  name: Path<TFieldValues>
  lineNumbersRef: React.RefObject<HTMLDivElement | null>
  textareaRef: React.RefObject<HTMLTextAreaElement | null>
  onScroll: (event: React.UIEvent<HTMLTextAreaElement>) => void
  isPending: boolean
  targetValidation: {
    count: number
    invalid: Array<{ index: number; lineNumber: number; originalTarget: string; error: string }>
  }
}

export function LinkTargetInputSection<TFieldValues extends FieldValues>({
  t,
  formControl,
  name,
  lineNumbersRef,
  textareaRef,
  onScroll,
  isPending,
  targetValidation,
}: LinkTargetInputSectionProps<TFieldValues>) {
  return (
    <FormField
      control={formControl}
      name={name}
      render={({ field }) => {
        const fieldValue = typeof field.value === "string" ? field.value : ""
        const lineIssues: BulkLineValidationIssue[] = targetValidation.invalid.map((target) => ({
          id: `invalid-${target.index}`,
          lineNumber: target.lineNumber,
          tone: "error",
          badgeLabel: t("invalidBadge"),
          message: t("invalidIssue", {
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
          <FormItem>
            <BulkLineValidationInput
              id="organization-link-targets"
              name={field.name}
              label={t("targetLabel")}
              required
              placeholder={t("placeholder")}
              value={fieldValue}
              lineCount={Math.max(countLineNumberedTextareaLines(fieldValue), 15)}
              lineNumbersRef={lineNumbersRef}
              textareaRef={textareaRef}
              onTextareaRef={field.ref}
              onValueChange={field.onChange}
              onScroll={onScroll}
              disabled={isPending}
              helper={t("targetHelper")}
              example={t("targetExample")}
              emptySummary={t("emptySummary")}
              validSummary={t("validSummary", { count: targetValidation.count })}
              blockingSummary={t("blockingSummary", { count: targetValidation.invalid.length })}
              collapseDetails={t("collapseDetails")}
              expandDetails={t("expandDetails")}
              validationResult={validationResult}
              viewportClassName={lineNumberedTextareaResponsiveViewportClassName}
            />
            <FormMessage />
          </FormItem>
        )
      }}
    />
  )
}

interface LinkTargetOrganizationSectionProps {
  organizationName: string
  t: TranslationFn
}

export function LinkTargetOrganizationSection({ organizationName, t }: LinkTargetOrganizationSectionProps) {
  return (
    <div className="gap-2 grid">
      <Label>{t("organizationLabel")}</Label>
      <div className="bg-muted/50 border flex gap-2 items-center px-3 py-2 rounded-md">
        <OrganizationIcon className="h-4 text-muted-foreground w-4" />
        <span className="font-medium">{organizationName}</span>
      </div>
    </div>
  )
}

interface LinkTargetDialogFooterProps {
  t: TranslationFn
  isPending: boolean
  isFormValid: boolean
}

export function LinkTargetDialogFooter({
  t,
  isPending,
  isFormValid,
}: LinkTargetDialogFooterProps) {
  return (
    <div className="flex justify-end gap-2">
      <Button
        type="submit"
        disabled={isPending || !isFormValid}
        loading={isPending}
        loadingLabel={t("creating")}
      >
        <>
          <AddIcon />
          {t("createTarget")}
        </>
      </Button>
    </div>
  )
}
