"use client"

import React from "react"
import type {
  FieldArrayWithId,
  FieldErrors,
  UseFormRegister,
  UseFormSetValue,
} from "react-hook-form"
import { IconPlus, IconTrash } from "@/components/icons"
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
import type { GobyRule } from "@/types/fingerprint.types"
import type { GobyFingerprintFormData } from "@/components/fingerprints/goby-fingerprint-dialog-state"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

type Option = { value: string; label: string }

type RuleField = FieldArrayWithId<GobyFingerprintFormData, "rule", "id">

type RuleAppend = (value: GobyRule) => void

type RuleRemove = (index: number) => void

const LABEL_OPTIONS: Option[] = [
  { value: "title", label: "title" },
  { value: "header", label: "header" },
  { value: "body", label: "body" },
  { value: "server", label: "server" },
  { value: "banner", label: "banner" },
  { value: "port", label: "port" },
  { value: "protocol", label: "protocol" },
  { value: "cert", label: "cert" },
]

interface GobyFingerprintDialogHeaderProps {
  t: TranslationFn
  isEdit: boolean
}

export function GobyFingerprintDialogHeader({ t, isEdit }: GobyFingerprintDialogHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle>{isEdit ? t("goby.editTitle") : t("goby.addTitle")}</DialogTitle>
      <DialogDescription>
        {isEdit ? t("goby.editDesc") : t("goby.addDesc")}
      </DialogDescription>
    </DialogHeader>
  )
}

interface GobyFingerprintBasicFieldsProps {
  t: TranslationFn
  tColumns: TranslationFn
  register: UseFormRegister<GobyFingerprintFormData>
  errors: FieldErrors<GobyFingerprintFormData>
}

export function GobyFingerprintBasicFields({
  t,
  tColumns,
  register,
  errors,
}: GobyFingerprintBasicFieldsProps) {
  return (
    <div className="grid grid-cols-2 gap-4">
      <div className="space-y-2">
        <Label htmlFor="name">{tColumns("name") || "Name"} *</Label>
        <Input autoComplete="off"
          id="name"
          placeholder={t("form.namePlaceholder")}
          {...register("name", { required: t("form.nameRequired") })}
        />
        {errors.name && (
          <p className="text-sm text-destructive">{errors.name.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="logic">{tColumns("logic")} *</Label>
        <Input autoComplete="off"
          id="logic"
          placeholder={t("form.logicPlaceholder")}
          {...register("logic", { required: t("form.logicRequired") })}
        />
        {errors.logic && (
          <p className="text-sm text-destructive">{errors.logic.message}</p>
        )}
      </div>
    </div>
  )
}

interface GobyFingerprintRulesSectionProps {
  t: TranslationFn
  tCommon: TranslationFn
  tColumns: TranslationFn
  fields: RuleField[]
  register: UseFormRegister<GobyFingerprintFormData>
  setValue: UseFormSetValue<GobyFingerprintFormData>
  watchedRules: GobyRule[]
  append: RuleAppend
  remove: RuleRemove
}

export function GobyFingerprintRulesSection({
  t,
  tCommon,
  tColumns,
  fields,
  register,
  setValue,
  watchedRules,
  append,
  remove,
}: GobyFingerprintRulesSectionProps) {
  const handleAddRule = () => {
    append({ label: "title", feature: "", is_equal: true })
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <Label>{tColumns("rules")} *</Label>
        <Button type="button" variant="outline" size="sm" onClick={handleAddRule}>
          <IconPlus className="h-4 w-4" />
          {tCommon("add")}
        </Button>
      </div>

      <div className="space-y-2">
        {fields.map((field, index) => (
          <div key={field.id} className="flex items-center gap-2 p-3 border rounded-md bg-muted/30">
            <div className="w-24">
              <Select
                value={watchedRules[index]?.label || "title"}
                onValueChange={(value) => setValue(`rule.${index}.label`, value)}
              >
                <SelectTrigger className="h-8">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {LABEL_OPTIONS.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex-1">
              <Input autoComplete="off"
                {...register(`rule.${index}.feature` as const, { required: true })}
                placeholder={t("form.featurePlaceholder")}
                className="h-8"
              />
            </div>
            <div className="flex items-center gap-1">
              <Checkbox
                checked={watchedRules[index]?.is_equal ?? true}
                onCheckedChange={(checked) => setValue(`rule.${index}.is_equal`, !!checked)}
              />
              <span className="text-xs text-muted-foreground">Match</span>
            </div>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => remove(index)}
              disabled={fields.length <= 1}
              className="h-8 w-8 p-0"
            >
              <IconTrash className="h-4 w-4 text-destructive" />
            </Button>
          </div>
        ))}
      </div>
    </div>
  )
}

interface GobyFingerprintDialogFooterProps {
  tCommon: TranslationFn
  isEdit: boolean
  isSubmitting: boolean
  onCancel: () => void
}

export function GobyFingerprintDialogFooter({
  tCommon,
  isEdit,
  isSubmitting,
  onCancel,
}: GobyFingerprintDialogFooterProps) {
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
