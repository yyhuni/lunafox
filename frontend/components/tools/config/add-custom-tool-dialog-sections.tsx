"use client"

import React from "react"
import { Wrench } from "@/components/icons"
import { IconPlus } from "@/components/icons"
import { IconX } from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { LoadingSpinner } from "@/components/loading-spinner"
import { CategoryNameMap } from "@/types/tool.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface AddCustomToolDialogHeaderProps {
  t: TranslationFn
  isEditMode: boolean
}

export function AddCustomToolDialogHeader({ t, isEditMode }: AddCustomToolDialogHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle className="flex items-center space-x-2">
        <Wrench />
        <span>{isEditMode ? t("editCustomTool") : t("addCustomTool")}</span>
      </DialogTitle>
      <DialogDescription>{t("customDialogDesc")}</DialogDescription>
    </DialogHeader>
  )
}

interface AddCustomToolNameFieldProps {
  t: TranslationFn
  value: string
  isSubmitting: boolean
  onChange: (value: string) => void
}

export function AddCustomToolNameField({
  t,
  value,
  isSubmitting,
  onChange,
}: AddCustomToolNameFieldProps) {
  return (
    <div className="grid gap-2">
      <Label htmlFor="name">
        {t("toolName")} <span className="text-red-500">*</span>
      </Label>
      <Input
        id="name"
        name="name"
        autoComplete="off"
        placeholder={t("customToolNamePlaceholder")}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        disabled={isSubmitting}
        required
      />
    </div>
  )
}

interface AddCustomToolDescriptionFieldProps {
  t: TranslationFn
  value: string
  isSubmitting: boolean
  onChange: (value: string) => void
}

export function AddCustomToolDescriptionField({
  t,
  value,
  isSubmitting,
  onChange,
}: AddCustomToolDescriptionFieldProps) {
  return (
    <div className="grid gap-2">
      <Label htmlFor="description">{t("toolDesc")}</Label>
      <Textarea
        id="description"
        name="description"
        autoComplete="off"
        placeholder={t("customToolDescPlaceholder")}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        disabled={isSubmitting}
        rows={3}
      />
    </div>
  )
}

interface AddCustomToolPathFieldProps {
  t: TranslationFn
  value: string
  isSubmitting: boolean
  onChange: (value: string) => void
}

export function AddCustomToolPathField({
  t,
  value,
  isSubmitting,
  onChange,
}: AddCustomToolPathFieldProps) {
  return (
    <div className="grid gap-2">
      <Label htmlFor="directory">
        {t("toolPath")} <span className="text-red-500">*</span>
      </Label>
      <Input
        id="directory"
        name="directory"
        autoComplete="off"
        placeholder={t("toolPathPlaceholder")}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        disabled={isSubmitting}
        required
      />
      <p className="text-xs text-muted-foreground">{t("toolPathHint")}</p>
    </div>
  )
}

interface AddCustomToolCategoriesFieldProps {
  t: TranslationFn
  isSubmitting: boolean
  availableCategories: string[]
  selectedCategories: string[]
  onToggle: (categoryName: string) => void
  onRemove: (categoryName: string) => void
}

export function AddCustomToolCategoriesField({
  t,
  isSubmitting,
  availableCategories,
  selectedCategories,
  onToggle,
  onRemove,
}: AddCustomToolCategoriesFieldProps) {
  return (
    <div className="grid gap-2">
      <Label>{t("categoryTags")}</Label>

      {selectedCategories.length > 0 && (
        <div className="flex flex-wrap gap-2 p-3 border rounded-md bg-muted/50">
          {selectedCategories.map((categoryName) => (
            <Badge
              key={categoryName}
              variant="default"
              className="flex items-center gap-1 px-2 py-1"
            >
              {CategoryNameMap[categoryName] || categoryName}
              <button
                type="button"
                onClick={() => onRemove(categoryName)}
                disabled={isSubmitting}
                className="ml-1 hover:bg-primary/20 rounded-full p-0.5"
              >
                <IconX className="h-3 w-3" />
              </button>
            </Badge>
          ))}
        </div>
      )}

      <div className="flex flex-wrap gap-2 p-3 border rounded-md">
        {availableCategories.length > 0 ? (
          availableCategories.map((categoryName) => {
            const isSelected = selectedCategories.includes(categoryName)
            return (
              <Badge
                asChild
                key={categoryName}
                variant={isSelected ? "secondary" : "outline"}
                className="hover:bg-secondary/80 transition-colors"
              >
                <button
                  type="button"
                  onClick={() => onToggle(categoryName)}
                >
                  {CategoryNameMap[categoryName] || categoryName}
                </button>
              </Badge>
            )
          })
        ) : (
          <p className="text-sm text-muted-foreground">{t("noCategories")}</p>
        )}
      </div>
    </div>
  )
}

interface AddCustomToolFooterProps {
  t: TranslationFn
  isEditMode: boolean
  isSubmitting: boolean
  isFormValid: boolean
  onCancel: () => void
}

export function AddCustomToolFooter({
  t,
  isEditMode,
  isSubmitting,
  isFormValid,
  onCancel,
}: AddCustomToolFooterProps) {
  return (
    <DialogFooter>
      <Button
        type="button"
        variant="outline"
        onClick={onCancel}
        disabled={isSubmitting}
      >
        {t("cancel")}
      </Button>
      <Button type="submit" disabled={isSubmitting || !isFormValid}>
        {isSubmitting ? (
          <>
            <LoadingSpinner />
            {isEditMode ? t("saving") : t("creating")}
          </>
        ) : (
          <>
            <IconPlus className="h-5 w-5" />
            {isEditMode ? t("saveChanges") : t("createTool")}
          </>
        )}
      </Button>
    </DialogFooter>
  )
}
