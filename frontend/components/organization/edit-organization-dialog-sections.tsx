"use client"

import React from "react"
import type { Control, FieldValues, Path } from "react-hook-form"
import { semanticIcons } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

const EditIcon = semanticIcons.action.edit

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

interface EditOrganizationChangesNoticeProps {
  t: TranslationFn
}

export function EditOrganizationChangesNotice({ t }: EditOrganizationChangesNoticeProps) {
  return (
    <div className="bg-warning/10 p-2 rounded text-warning text-xs">
      {t("changesDetected")}
    </div>
  )
}

interface EditOrganizationFooterProps {
  t: TranslationFn
  isUpdating: boolean
  isFormValid: boolean
  hasChanges: boolean
  onReset: () => void
}

export function EditOrganizationFooter({
  t,
  isUpdating,
  isFormValid,
  hasChanges,
  onReset,
}: EditOrganizationFooterProps) {
  return (
    <div className="flex flex-wrap justify-end gap-2">
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
        loading={isUpdating}
        loadingLabel={t("updating")}
      >
        <>
          <EditIcon />
          {t("update")}
        </>
      </Button>
    </div>
  )
}
