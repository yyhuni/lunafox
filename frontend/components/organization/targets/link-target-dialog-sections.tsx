"use client"

import React from "react"
import type { Control, FieldValues, Path } from "react-hook-form"
import { Building2, Plus, Target } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { LoadingSpinner } from "@/components/loading-spinner"
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface LinkTargetDialogHeaderProps {
  organizationName: string
  t: TranslationFn
}

export function LinkTargetDialogHeader({ organizationName, t }: LinkTargetDialogHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle className="flex items-center space-x-2">
        <Target />
        <span>{t("title")}</span>
      </DialogTitle>
      <DialogDescription>
        {t("description", { name: organizationName })}
      </DialogDescription>
    </DialogHeader>
  )
}

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
    invalid: Array<{ index: number; originalTarget: string; error: string }>
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
      render={({ field }) => (
        <FormItem>
          <FormLabel>
            {t("targetLabel")} <span className="text-destructive">*</span>
          </FormLabel>
          <FormControl>
            <div className="relative border rounded-md overflow-hidden bg-background">
              <div className="flex h-[324px]">
                <div className="flex-shrink-0 w-12 bg-muted/30 border-r select-none overflow-hidden">
                  <div
                    ref={lineNumbersRef}
                    className="py-3 px-2 text-right font-mono text-xs text-muted-foreground leading-[1.4] h-full overflow-y-auto scrollbar-hide"
                  >
                    {Array.from({ length: Math.max(field.value.split("\n").length, 15) }, (_, i) => (
                      <div key={i + 1} className="h-[20px]">
                        {i + 1}
                      </div>
                    ))}
                  </div>
                </div>
                <Textarea
                  {...field}
                  ref={(element) => {
                    field.ref(element)
                    textareaRef.current = element
                  }}
                  onScroll={onScroll}
                  placeholder={t("placeholder")}
                  disabled={isPending}
                  autoComplete="off"
                  className="font-mono h-full overflow-y-auto resize-none border-0 focus-visible:ring-0 focus-visible:ring-offset-0 leading-[1.4] text-sm py-3"
                  style={{ lineHeight: "20px" }}
                />
              </div>
            </div>
          </FormControl>
          <FormDescription>
            {t("targetCount", { count: targetValidation.count })}
            {targetValidation.invalid.length > 0 && (
              <span className="text-destructive ml-2">
                | {t("invalidCount", { count: targetValidation.invalid.length })}
              </span>
            )}
          </FormDescription>
          {targetValidation.invalid.length > 0 && (
            <div className="text-xs text-destructive">
              {t("invalidExample", {
                line: targetValidation.invalid[0].index + 1,
                target: targetValidation.invalid[0].originalTarget,
                error: targetValidation.invalid[0].error,
              })}
            </div>
          )}
          <FormMessage />
        </FormItem>
      )}
    />
  )
}

interface LinkTargetOrganizationSectionProps {
  organizationName: string
  t: TranslationFn
}

export function LinkTargetOrganizationSection({ organizationName, t }: LinkTargetOrganizationSectionProps) {
  return (
    <div className="grid gap-2">
      <Label className="flex items-center space-x-2">
        <Building2 />
        <span>{t("organizationLabel")}</span>
      </Label>
      <div className="flex items-center gap-2 px-3 py-2 border rounded-md bg-muted/50">
        <Building2 className="h-4 w-4 text-muted-foreground" />
        <span className="font-medium">{organizationName}</span>
      </div>
    </div>
  )
}

interface LinkTargetDialogFooterProps {
  tCommon: TranslationFn
  t: TranslationFn
  onCancel: () => void
  isPending: boolean
  isFormValid: boolean
}

export function LinkTargetDialogFooter({
  tCommon,
  t,
  onCancel,
  isPending,
  isFormValid,
}: LinkTargetDialogFooterProps) {
  return (
    <DialogFooter>
      <Button
        type="button"
        variant="outline"
        onClick={onCancel}
        disabled={isPending}
      >
        {tCommon("actions.cancel")}
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
            <Plus />
            {t("createTarget")}
          </>
        )}
      </Button>
    </DialogFooter>
  )
}
