"use client"

import React from "react"
import type { Control, FieldValues, Path } from "react-hook-form"
import { Building2, Plus, Target } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
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

interface AddOrganizationHeaderProps {
  t: TranslationFn
}

export function AddOrganizationHeader({ t }: AddOrganizationHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle className="flex items-center space-x-2">
        <Building2 />
        <span>{t("addTitle")}</span>
      </DialogTitle>
      <DialogDescription>{t("addDesc")}</DialogDescription>
    </DialogHeader>
  )
}

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
          <FormLabel>{t("orgDesc")}</FormLabel>
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
  targetValidation: {
    count: number
    invalid: Array<{ index: number; originalTarget: string; error: string }>
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
  targetValidation,
  name,
}: AddOrganizationTargetsFieldProps<TFieldValues>) {
  return (
    <FormField
      control={formControl}
      name={name}
      render={({ field }) => (
        <FormItem>
          <FormLabel className="flex items-center space-x-2">
            <Target className="h-4 w-4" />
            <span>{t("addTargets")}</span>
          </FormLabel>
          <FormControl>
            <div className="relative border rounded-md overflow-hidden bg-background">
              <div className="flex h-[324px]">
                <div className="flex-shrink-0 w-12 bg-muted/30 border-r select-none overflow-hidden">
                  <div
                    ref={lineNumbersRef}
                    className="py-3 px-2 text-right font-mono text-xs text-muted-foreground leading-[1.4] h-full overflow-y-auto scrollbar-hide"
                  >
                    {Array.from({ length: Math.max(field.value?.split("\n").length || 1, 15) }, (_, i) => (
                      <div key={i + 1} className="h-[20px]">{i + 1}</div>
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
                  placeholder={t("targetsPlaceholder")}
                  disabled={isSubmitting}
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

interface AddOrganizationFooterProps {
  t: TranslationFn
  isSubmitting: boolean
  isFormValid: boolean
  createPending: boolean
  onCancel: () => void
}

export function AddOrganizationFooter({
  t,
  isSubmitting,
  isFormValid,
  createPending,
  onCancel,
}: AddOrganizationFooterProps) {
  return (
    <DialogFooter>
      <Button type="button" variant="outline" onClick={onCancel} disabled={isSubmitting}>
        {t("cancel")}
      </Button>
      <Button type="submit" disabled={isSubmitting || !isFormValid}>
        {isSubmitting ? (
          <>
            <LoadingSpinner />
            {createPending ? t("creating") : t("creatingTargets")}
          </>
        ) : (
          <>
            <Plus />
            {t("create")}
          </>
        )}
      </Button>
    </DialogFooter>
  )
}
