"use client"

import React from "react"
import type { Control, FieldValues, Path } from "react-hook-form"
import { Building2, Edit } from "@/components/icons"
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

interface EditOrganizationHeaderProps {
  t: TranslationFn
}

export function EditOrganizationHeader({ t }: EditOrganizationHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle className="flex items-center space-x-2">
        <Building2 />
        <span>{t("editTitle")}</span>
      </DialogTitle>
      <DialogDescription>{t("editDesc")}</DialogDescription>
    </DialogHeader>
  )
}

interface EditOrganizationNameFieldProps<TFieldValues extends FieldValues> {
  t: TranslationFn
  formControl: Control<TFieldValues>
  isSubmitting: boolean
  name: Path<TFieldValues>
}

export function EditOrganizationNameField<TFieldValues extends FieldValues>({
  t,
  formControl,
  isSubmitting,
  name,
}: EditOrganizationNameFieldProps<TFieldValues>) {
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

interface EditOrganizationDescriptionFieldProps<TFieldValues extends FieldValues> {
  t: TranslationFn
  formControl: Control<TFieldValues>
  isSubmitting: boolean
  name: Path<TFieldValues>
}

export function EditOrganizationDescriptionField<TFieldValues extends FieldValues>({
  t,
  formControl,
  isSubmitting,
  name,
}: EditOrganizationDescriptionFieldProps<TFieldValues>) {
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

interface EditOrganizationChangesNoticeProps {
  t: TranslationFn
}

export function EditOrganizationChangesNotice({ t }: EditOrganizationChangesNoticeProps) {
  return (
    <div className="text-xs text-amber-600 bg-amber-50 dark:bg-amber-950/20 p-2 rounded">
      {t("changesDetected")}
    </div>
  )
}

interface EditOrganizationFooterProps {
  t: TranslationFn
  isUpdating: boolean
  isFormValid: boolean
  hasChanges: boolean
  onCancel: () => void
  onReset: () => void
}

export function EditOrganizationFooter({
  t,
  isUpdating,
  isFormValid,
  hasChanges,
  onCancel,
  onReset,
}: EditOrganizationFooterProps) {
  return (
    <DialogFooter className="gap-2">
      <Button
        type="button"
        variant="outline"
        onClick={onCancel}
        disabled={isUpdating}
      >
        {t("cancel")}
      </Button>

      {hasChanges && (
        <Button
          type="button"
          variant="ghost"
          onClick={onReset}
          disabled={isUpdating}
        >
          {t("reset")}
        </Button>
      )}

      <Button
        type="submit"
        disabled={isUpdating || !isFormValid || !hasChanges}
      >
        {isUpdating ? (
          <>
            <LoadingSpinner />
            {t("updating")}
          </>
        ) : (
          <>
            <Edit />
            {t("update")}
          </>
        )}
      </Button>
    </DialogFooter>
  )
}
