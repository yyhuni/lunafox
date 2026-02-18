"use client"

import React from "react"
import type { FieldErrors, UseFormRegister } from "react-hook-form"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import type { WappalyzerFingerprintFormData } from "@/components/fingerprints/wappalyzer-fingerprint-dialog-state"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface WappalyzerFingerprintDialogHeaderProps {
  t: TranslationFn
  isEdit: boolean
}

export function WappalyzerFingerprintDialogHeader({ t, isEdit }: WappalyzerFingerprintDialogHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle>{isEdit ? t("wappalyzer.editTitle") : t("wappalyzer.addTitle")}</DialogTitle>
      <DialogDescription>
        {isEdit ? t("wappalyzer.editDesc") : t("wappalyzer.addDesc")}
      </DialogDescription>
    </DialogHeader>
  )
}

interface WappalyzerFingerprintBasicFieldsProps {
  t: TranslationFn
  tCommon: TranslationFn
  register: UseFormRegister<WappalyzerFingerprintFormData>
  errors: FieldErrors<WappalyzerFingerprintFormData>
}

export function WappalyzerFingerprintBasicFields({
  t,
  tCommon,
  register,
  errors,
}: WappalyzerFingerprintBasicFieldsProps) {
  return (
    <>
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="name">{t("form.appNamePlaceholder").split("：")[0]} *</Label>
          <Input autoComplete="off"
            id="name"
            placeholder={t("form.appNamePlaceholder")}
            {...register("name", { required: t("form.appNameRequired") })}
          />
          {errors.name && (
            <p className="text-sm text-destructive">{errors.name.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="cats">{t("category")}</Label>
          <Input autoComplete="off"
            id="cats"
            placeholder={t("form.catsPlaceholder")}
            {...register("cats")}
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="website">{tCommon("website")}</Label>
          <Input autoComplete="off"
            id="website"
            placeholder="https://example.com"
            {...register("website")}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="cpe">CPE</Label>
          <Input autoComplete="off"
            id="cpe"
            placeholder="cpe:/a:vendor:product"
            {...register("cpe")}
          />
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="description">{tCommon("description")}</Label>
        <Textarea autoComplete="off"
          id="description"
          placeholder={t("form.descPlaceholder")}
          rows={2}
          {...register("description")}
        />
      </div>
    </>
  )
}

interface WappalyzerFingerprintDetectionFieldsProps {
  t: TranslationFn
  register: UseFormRegister<WappalyzerFingerprintFormData>
}

export function WappalyzerFingerprintDetectionFields({ t, register }: WappalyzerFingerprintDetectionFieldsProps) {
  return (
    <>
      <div className="space-y-1">
        <Label className="text-sm font-medium">{t("form.detectionRules")}</Label>
        <p className="text-xs text-muted-foreground">{t("form.detectionRulesHint")}</p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="cookies">{t("form.cookies")}</Label>
          <Textarea autoComplete="off"
            id="cookies"
            placeholder='{"name": "pattern"}'
            rows={2}
            className="font-mono text-xs"
            {...register("cookies")}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="headers">{t("form.headers")}</Label>
          <Textarea autoComplete="off"
            id="headers"
            placeholder='{"X-Powered-By": "pattern"}'
            rows={2}
            className="font-mono text-xs"
            {...register("headers")}
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="scriptSrc">{t("form.scriptUrl")}</Label>
          <Input autoComplete="off"
            id="scriptSrc"
            placeholder="pattern1, pattern2"
            className="font-mono text-xs"
            {...register("scriptSrc")}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="js">{t("form.jsVariables")}</Label>
          <Input autoComplete="off"
            id="js"
            placeholder="window.var1, window.var2"
            className="font-mono text-xs"
            {...register("js")}
          />
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="meta">{t("form.metaTags")} (JSON)</Label>
        <Textarea autoComplete="off"
          id="meta"
          placeholder='{"generator": ["pattern"]}'
          rows={2}
          className="font-mono text-xs"
          {...register("meta")}
        />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="html">{t("form.htmlContent")}</Label>
          <Input autoComplete="off"
            id="html"
            placeholder="pattern1, pattern2"
            className="font-mono text-xs"
            {...register("html")}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="implies">{t("form.implies")}</Label>
          <Input autoComplete="off"
            id="implies"
            placeholder="PHP, MySQL"
            {...register("implies")}
          />
        </div>
      </div>
    </>
  )
}

interface WappalyzerFingerprintDialogFooterProps {
  tCommon: TranslationFn
  isEdit: boolean
  isSubmitting: boolean
  onCancel: () => void
}

export function WappalyzerFingerprintDialogFooter({
  tCommon,
  isEdit,
  isSubmitting,
  onCancel,
}: WappalyzerFingerprintDialogFooterProps) {
  return (
    <DialogFooter>
      <Button type="button" variant="outline" onClick={onCancel}>
        {tCommon("cancel")}
      </Button>
      <Button type="submit" disabled={isSubmitting}>
        {isSubmitting ? tCommon("saving") : isEdit ? tCommon("update") : tCommon("create")}
      </Button>
    </DialogFooter>
  )
}
