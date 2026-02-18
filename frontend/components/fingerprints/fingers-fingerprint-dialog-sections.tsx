"use client"

import React from "react"
import type { FieldErrors, UseFormRegister } from "react-hook-form"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Textarea } from "@/components/ui/textarea"
import type { FingersFingerprintFormData } from "@/components/fingerprints/fingers-fingerprint-dialog-state"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface FingersFingerprintDialogHeaderProps {
  t: TranslationFn
  isEdit: boolean
}

export function FingersFingerprintDialogHeader({ t, isEdit }: FingersFingerprintDialogHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle>{isEdit ? t("fingers.editTitle") : t("fingers.addTitle")}</DialogTitle>
      <DialogDescription>
        {isEdit ? t("fingers.editDesc") : t("fingers.addDesc")}
      </DialogDescription>
    </DialogHeader>
  )
}

interface FingersFingerprintNameFieldProps {
  t: TranslationFn
  register: UseFormRegister<FingersFingerprintFormData>
  errors: FieldErrors<FingersFingerprintFormData>
}

export function FingersFingerprintNameField({
  t,
  register,
  errors,
}: FingersFingerprintNameFieldProps) {
  return (
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
  )
}

interface FingersFingerprintLinkFieldProps {
  t: TranslationFn
  register: UseFormRegister<FingersFingerprintFormData>
}

export function FingersFingerprintLinkField({
  t,
  register,
}: FingersFingerprintLinkFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor="link">{t("form.link")}</Label>
      <Input autoComplete="off"
        id="link"
        placeholder={t("form.linkPlaceholder")}
        {...register("link")}
      />
    </div>
  )
}

interface FingersFingerprintRuleFieldProps {
  t: TranslationFn
  register: UseFormRegister<FingersFingerprintFormData>
  errors: FieldErrors<FingersFingerprintFormData>
}

export function FingersFingerprintRuleField({
  t,
  register,
  errors,
}: FingersFingerprintRuleFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor="rule">{t("form.rule")} *</Label>
      <Textarea autoComplete="off"
        id="rule"
        placeholder={t("form.rulePlaceholder")}
        className="font-mono text-sm min-h-[120px]"
        {...register("rule", { required: t("form.ruleRequired") })}
      />
      {errors.rule && (
        <p className="text-sm text-destructive">{errors.rule.message}</p>
      )}
    </div>
  )
}

interface FingersFingerprintTagsFocusProps {
  t: TranslationFn
  register: UseFormRegister<FingersFingerprintFormData>
  focus: boolean
  onFocusChange: (value: boolean) => void
}

export function FingersFingerprintTagsFocus({
  t,
  register,
  focus,
  onFocusChange,
}: FingersFingerprintTagsFocusProps) {
  return (
    <div className="grid grid-cols-2 gap-4">
      <div className="space-y-2">
        <Label htmlFor="tag">{t("form.tag")}</Label>
        <Input autoComplete="off"
          id="tag"
          placeholder={t("form.tagPlaceholder")}
          {...register("tag")}
        />
      </div>

      <div className="space-y-2">
        <Label>{t("form.mark")}</Label>
        <div className="flex items-center space-x-2 h-9">
          <Checkbox
            id="focus"
            checked={focus}
            onCheckedChange={(checked) => onFocusChange(!!checked)}
          />
          <Label htmlFor="focus" className="cursor-pointer font-normal">
            {t("form.focusLabel")}
          </Label>
        </div>
      </div>
    </div>
  )
}

interface FingersFingerprintDefaultPortProps {
  t: TranslationFn
  register: UseFormRegister<FingersFingerprintFormData>
}

export function FingersFingerprintDefaultPort({
  t,
  register,
}: FingersFingerprintDefaultPortProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor="defaultPort">{t("form.defaultPort")}</Label>
      <Input autoComplete="off"
        id="defaultPort"
        placeholder={t("form.defaultPortPlaceholder")}
        {...register("defaultPort")}
      />
    </div>
  )
}

interface FingersFingerprintDialogFooterProps {
  tCommon: TranslationFn
  isEdit: boolean
  isSubmitting: boolean
  onCancel: () => void
}

export function FingersFingerprintDialogFooter({
  tCommon,
  isEdit,
  isSubmitting,
  onCancel,
}: FingersFingerprintDialogFooterProps) {
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
