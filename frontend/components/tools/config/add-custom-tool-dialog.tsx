"use client"

import React from "react"
import { IconPlus } from "@/components/icons"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogTrigger,
} from "@/components/ui/dialog"

import { useAddCustomToolDialogState } from "@/components/tools/config/add-custom-tool-dialog-state"
import {
  AddCustomToolDialogHeader,
  AddCustomToolNameField,
  AddCustomToolDescriptionField,
  AddCustomToolPathField,
  AddCustomToolCategoriesField,
  AddCustomToolFooter,
} from "@/components/tools/config/add-custom-tool-dialog-sections"
import type { Tool } from "@/types/tool.types"

interface AddCustomToolDialogProps {
  tool?: Tool
  onAdd?: (tool: Tool) => void
  open?: boolean
  onOpenChange?: (open: boolean) => void
}

export function AddCustomToolDialog({
  tool,
  onAdd,
  open: externalOpen,
  onOpenChange: externalOnOpenChange,
}: AddCustomToolDialogProps) {
  const t = useTranslations("tools.config")

  const {
    open,
    formData,
    isEditMode,
    availableCategories,
    isFormValid,
    isSubmitting,
    handleOpenChange,
    handleCategoryToggle,
    handleCategoryRemove,
    handleSubmit,
    setFormField,
  } = useAddCustomToolDialogState({
    tool,
    onAdd,
    open: externalOpen,
    onOpenChange: externalOnOpenChange,
  })

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      {externalOpen === undefined && (
        <DialogTrigger asChild>
          <Button>
            <IconPlus className="h-5 w-5" />
            {t("addTool")}
          </Button>
        </DialogTrigger>
      )}

      <DialogContent className="sm:max-w-[500px]">
        <AddCustomToolDialogHeader t={t} isEditMode={isEditMode} />

        <form onSubmit={handleSubmit}>
          <div className="grid gap-6 py-4">
            <AddCustomToolNameField
              t={t}
              value={formData.name}
              isSubmitting={isSubmitting}
              onChange={(value) => setFormField("name", value)}
            />
            <AddCustomToolDescriptionField
              t={t}
              value={formData.description}
              isSubmitting={isSubmitting}
              onChange={(value) => setFormField("description", value)}
            />
            <AddCustomToolPathField
              t={t}
              value={formData.directory}
              isSubmitting={isSubmitting}
              onChange={(value) => setFormField("directory", value)}
            />
            <AddCustomToolCategoriesField
              t={t}
              isSubmitting={isSubmitting}
              availableCategories={availableCategories}
              selectedCategories={formData.categoryNames}
              onToggle={handleCategoryToggle}
              onRemove={handleCategoryRemove}
            />
          </div>

          <AddCustomToolFooter
            t={t}
            isEditMode={isEditMode}
            isSubmitting={isSubmitting}
            isFormValid={isFormValid}
            onCancel={() => handleOpenChange(false)}
          />
        </form>
      </DialogContent>
    </Dialog>
  )
}
