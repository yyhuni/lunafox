"use client"

import React from "react"
import type { FieldErrors, UseFormRegister } from "react-hook-form"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import type { FingerPrintHubFingerprintFormData } from "@/components/fingerprints/fingerprinthub-fingerprint-dialog-state"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

type Option = { value: string; label: string }

const SEVERITY_OPTIONS: Option[] = [
  { value: "info", label: "Info" },
  { value: "low", label: "Low" },
  { value: "medium", label: "Medium" },
  { value: "high", label: "High" },
  { value: "critical", label: "Critical" },
]

interface FingerPrintHubDialogHeaderProps {
  t: TranslationFn
  isEdit: boolean
}

export function FingerPrintHubDialogHeader({ t, isEdit }: FingerPrintHubDialogHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle>{isEdit ? t("fingerprinthub.editTitle") : t("fingerprinthub.addTitle")}</DialogTitle>
      <DialogDescription>
        {isEdit ? t("fingerprinthub.editDesc") : t("fingerprinthub.addDesc")}
      </DialogDescription>
    </DialogHeader>
  )
}

interface FingerPrintHubIdNameFieldsProps {
  t: TranslationFn
  register: UseFormRegister<FingerPrintHubFingerprintFormData>
  errors: FieldErrors<FingerPrintHubFingerprintFormData>
}

export function FingerPrintHubIdNameFields({
  t,
  register,
  errors,
}: FingerPrintHubIdNameFieldsProps) {
  return (
    <div className="grid grid-cols-2 gap-4">
      <div className="space-y-2">
        <Label htmlFor="fpId">{t("form.fpId")} *</Label>
        <Input autoComplete="off"
          id="fpId"
          placeholder={t("form.fpIdPlaceholder")}
          {...register("fpId", { required: t("form.fpIdRequired") })}
        />
        {errors.fpId && (
          <p className="text-sm text-destructive">{errors.fpId.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="name">{t("form.name")} *</Label>
        <Input autoComplete="off"
          id="name"
          placeholder={t("form.namePlaceholder")}
          {...register("name", { required: t("form.nameRequired") })}
        />
        {errors.name && (
          <p className="text-sm text-destructive">{errors.name.message}</p>
        )}
      </div>
    </div>
  )
}

interface FingerPrintHubAuthorSeverityFieldsProps {
  t: TranslationFn
  severity: string
  onSeverityChange: (value: string) => void
  register: UseFormRegister<FingerPrintHubFingerprintFormData>
}

export function FingerPrintHubAuthorSeverityFields({
  t,
  severity,
  onSeverityChange,
  register,
}: FingerPrintHubAuthorSeverityFieldsProps) {
  return (
    <div className="grid grid-cols-2 gap-4">
      <div className="space-y-2">
        <Label htmlFor="author">{t("form.author")}</Label>
        <Input autoComplete="off"
          id="author"
          placeholder={t("form.authorPlaceholder")}
          {...register("author")}
        />
      </div>

      <div className="space-y-2">
        <Label>{t("form.severity")}</Label>
        <Select value={severity} onValueChange={onSeverityChange}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {SEVERITY_OPTIONS.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}

interface FingerPrintHubTagsFieldProps {
  t: TranslationFn
  register: UseFormRegister<FingerPrintHubFingerprintFormData>
}

export function FingerPrintHubTagsField({
  t,
  register,
}: FingerPrintHubTagsFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor="tags">{t("form.tags")}</Label>
      <Input autoComplete="off"
        id="tags"
        placeholder={t("form.tagsPlaceholder")}
        {...register("tags")}
      />
    </div>
  )
}

interface FingerPrintHubMetadataFieldProps {
  t: TranslationFn
  register: UseFormRegister<FingerPrintHubFingerprintFormData>
}

export function FingerPrintHubMetadataField({
  t,
  register,
}: FingerPrintHubMetadataFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor="metadata">{t("form.metadata")} *</Label>
      <Textarea autoComplete="off"
        id="metadata"
        placeholder={t("form.metadataPlaceholder")}
        className="font-mono text-sm min-h-[120px]"
        {...register("metadata", { required: t("form.metadataRequired") })}
      />
    </div>
  )
}

interface FingerPrintHubHttpFieldProps {
  t: TranslationFn
  register: UseFormRegister<FingerPrintHubFingerprintFormData>
  errors: FieldErrors<FingerPrintHubFingerprintFormData>
}

export function FingerPrintHubHttpField({
  t,
  register,
  errors,
}: FingerPrintHubHttpFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor="http">{t("form.http")} *</Label>
      <Textarea autoComplete="off"
        id="http"
        placeholder={t("form.httpPlaceholder")}
        className="font-mono text-sm min-h-[150px]"
        {...register("http", { required: t("form.httpRequired") })}
      />
      {errors.http && (
        <p className="text-sm text-destructive">{errors.http.message}</p>
      )}
    </div>
  )
}

interface FingerPrintHubSourceFileFieldProps {
  t: TranslationFn
  register: UseFormRegister<FingerPrintHubFingerprintFormData>
}

export function FingerPrintHubSourceFileField({
  t,
  register,
}: FingerPrintHubSourceFileFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor="sourceFile">{t("form.sourceFile")}</Label>
      <Input autoComplete="off"
        id="sourceFile"
        placeholder={t("form.sourceFilePlaceholder")}
        {...register("sourceFile")}
      />
    </div>
  )
}

interface FingerPrintHubDialogFooterProps {
  tCommon: TranslationFn
  isEdit: boolean
  isSubmitting: boolean
  onCancel: () => void
}

export function FingerPrintHubDialogFooter({
  tCommon,
  isEdit,
  isSubmitting,
  onCancel,
}: FingerPrintHubDialogFooterProps) {
  return (
    <DialogFooter>
      <Button type="button" variant="outline" onClick={onCancel}>
        {tCommon("cancel")}
      </Button>
      <Button type="submit" disabled={isSubmitting}>
        {isSubmitting ? "…" : isEdit ? tCommon("save") : tCommon("create")}
      </Button>
    </DialogFooter>
  )
}
