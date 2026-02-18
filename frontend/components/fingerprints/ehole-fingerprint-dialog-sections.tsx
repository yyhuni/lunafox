"use client"

import React from "react"
import type { FieldErrors, UseFormRegister } from "react-hook-form"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import type { EholeFingerprintFormData } from "@/components/fingerprints/ehole-fingerprint-dialog-state"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

type Option = { value: string; label: string }

const METHOD_OPTIONS: Option[] = [
  { value: "keyword", label: "keyword" },
  { value: "faviconhash", label: "faviconhash" },
  { value: "icon_hash", label: "icon_hash" },
  { value: "header", label: "header" },
]

const LOCATION_OPTIONS: Option[] = [
  { value: "body", label: "body" },
  { value: "header", label: "header" },
  { value: "title", label: "title" },
]

interface EholeFingerprintDialogHeaderProps {
  t: TranslationFn
  isEdit: boolean
}

export function EholeFingerprintDialogHeader({ t, isEdit }: EholeFingerprintDialogHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle>{isEdit ? t("ehole.editTitle") : t("ehole.addTitle")}</DialogTitle>
      <DialogDescription>
        {isEdit ? t("ehole.editDesc") : t("ehole.addDesc")}
      </DialogDescription>
    </DialogHeader>
  )
}

interface EholeFingerprintCmsFieldProps {
  t: TranslationFn
  tColumns: TranslationFn
  register: UseFormRegister<EholeFingerprintFormData>
  errors: FieldErrors<EholeFingerprintFormData>
}

export function EholeFingerprintCmsField({
  t,
  tColumns,
  register,
  errors,
}: EholeFingerprintCmsFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor="cms">{tColumns("cms")} *</Label>
      <Input autoComplete="off"
        id="cms"
        placeholder={t("form.cmsPlaceholder")}
        {...register("cms", { required: t("form.cmsRequired") })}
      />
      {errors.cms && (
        <p className="text-sm text-destructive">{errors.cms.message}</p>
      )}
    </div>
  )
}

interface EholeFingerprintMethodLocationProps {
  t: TranslationFn
  tColumns: TranslationFn
  method: string
  location: string
  onMethodChange: (value: string) => void
  onLocationChange: (value: string) => void
}

export function EholeFingerprintMethodLocation({
  t,
  tColumns,
  method,
  location,
  onMethodChange,
  onLocationChange,
}: EholeFingerprintMethodLocationProps) {
  return (
    <div className="grid grid-cols-2 gap-4">
      <div className="space-y-2">
        <Label>{tColumns("method")} *</Label>
        <Select value={method} onValueChange={onMethodChange}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {METHOD_OPTIONS.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-2">
        <Label>{t("form.location")} *</Label>
        <Select value={location} onValueChange={onLocationChange}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {LOCATION_OPTIONS.map((option) => (
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

interface EholeFingerprintKeywordFieldProps {
  t: TranslationFn
  tColumns: TranslationFn
  register: UseFormRegister<EholeFingerprintFormData>
  errors: FieldErrors<EholeFingerprintFormData>
}

export function EholeFingerprintKeywordField({
  t,
  tColumns,
  register,
  errors,
}: EholeFingerprintKeywordFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor="keyword">{tColumns("keyword")} *</Label>
      <Input autoComplete="off"
        id="keyword"
        placeholder={t("form.keywordPlaceholder")}
        {...register("keyword", { required: t("form.keywordRequired") })}
      />
      {errors.keyword && (
        <p className="text-sm text-destructive">{errors.keyword.message}</p>
      )}
    </div>
  )
}

interface EholeFingerprintTypeImportantProps {
  t: TranslationFn
  tColumns: TranslationFn
  register: UseFormRegister<EholeFingerprintFormData>
  isImportant: boolean
  onImportantChange: (value: boolean) => void
}

export function EholeFingerprintTypeImportant({
  t,
  tColumns,
  register,
  isImportant,
  onImportantChange,
}: EholeFingerprintTypeImportantProps) {
  return (
    <div className="grid grid-cols-2 gap-4">
      <div className="space-y-2">
        <Label htmlFor="type">{tColumns("type") || "Type"}</Label>
        <Input autoComplete="off"
          id="type"
          placeholder={t("form.typePlaceholder")}
          {...register("type")}
        />
      </div>

      <div className="space-y-2">
        <Label>{t("form.mark")}</Label>
        <div className="flex items-center space-x-2 h-9">
          <Checkbox
            id="isImportant"
            checked={isImportant}
            onCheckedChange={(checked) => onImportantChange(!!checked)}
          />
          <Label htmlFor="isImportant" className="cursor-pointer font-normal">
            {tColumns("important")}
          </Label>
        </div>
      </div>
    </div>
  )
}

interface EholeFingerprintDialogFooterProps {
  tCommon: TranslationFn
  isEdit: boolean
  isSubmitting: boolean
  onCancel: () => void
}

export function EholeFingerprintDialogFooter({
  tCommon,
  isEdit,
  isSubmitting,
  onCancel,
}: EholeFingerprintDialogFooterProps) {
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
