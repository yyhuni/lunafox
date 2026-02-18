"use client"

import React from "react"
import type { FieldErrors, UseFormRegister } from "react-hook-form"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import type { ARLFingerprintFormData } from "@/components/fingerprints/arl-fingerprint-dialog-state"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface ARLFingerprintDialogHeaderProps {
  t: TranslationFn
  isEdit: boolean
}

export function ARLFingerprintDialogHeader({ t, isEdit }: ARLFingerprintDialogHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle>{isEdit ? t("arl.editTitle") : t("arl.addTitle")}</DialogTitle>
      <DialogDescription>
        {isEdit ? t("arl.editDesc") : t("arl.addDesc")}
      </DialogDescription>
    </DialogHeader>
  )
}

interface ARLFingerprintNameFieldProps {
  t: TranslationFn
  register: UseFormRegister<ARLFingerprintFormData>
  errors: FieldErrors<ARLFingerprintFormData>
}

export function ARLFingerprintNameField({
  t,
  register,
  errors,
}: ARLFingerprintNameFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor="name">{t("form.name")} *</Label>
      <Input autoComplete="off"
        id="name"
        placeholder={t("form.arlNamePlaceholder")}
        {...register("name", { required: t("form.nameRequired") })}
      />
      {errors.name && (
        <p className="text-sm text-destructive">{errors.name.message}</p>
      )}
    </div>
  )
}

interface ARLFingerprintRuleFieldProps {
  t: TranslationFn
  register: UseFormRegister<ARLFingerprintFormData>
  errors: FieldErrors<ARLFingerprintFormData>
}

export function ARLFingerprintRuleField({
  t,
  register,
  errors,
}: ARLFingerprintRuleFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor="rule">{t("form.arlRule")} *</Label>
      <Textarea autoComplete="off"
        id="rule"
        placeholder={t("form.arlRulePlaceholder")}
        className="font-mono text-sm min-h-[120px]"
        {...register("rule", { required: t("form.arlRuleRequired") })}
      />
      <p className="text-xs text-muted-foreground">
        {t("form.arlRuleHint")}
      </p>
      {errors.rule && (
        <p className="text-sm text-destructive">{errors.rule.message}</p>
      )}
    </div>
  )
}

interface ARLFingerprintDialogFooterProps {
  tCommon: TranslationFn
  isEdit: boolean
  isSubmitting: boolean
  onCancel: () => void
}

export function ARLFingerprintDialogFooter({
  tCommon,
  isEdit,
  isSubmitting,
  onCancel,
}: ARLFingerprintDialogFooterProps) {
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
