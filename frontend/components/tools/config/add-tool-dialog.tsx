"use client";
import React from "react";
import { IconPlus, semanticIcons } from "@/components/icons";
import { useTranslations } from "next-intl";
// Import UI components
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger, } from "@/components/ui/dialog";
import { Form } from "@/components/ui/form";
import { useAddToolDialogState } from "@/components/tools/config/add-tool-dialog-state";
import { AddToolBasicInfoSection, AddToolCommandSection, AddToolDialogFooter, } from "@/components/tools/config/add-tool-dialog-sections";
// Import type definitions
import type { Tool } from "@/types/tool.types";
// Component props type definition
interface AddToolDialogProps {
    tool?: Tool; // Tool data to edit (optional, edit mode when provided)
    onAdd?: (tool: Tool) => void; // Callback function on successful add (optional)
    open?: boolean; // External control for dialog open state
    onOpenChange?: (open: boolean) => void; // External control callback for dialog open state
}
const ToolIcon = semanticIcons.concept.tool;
/**
 * Add tool dialog component (using React Query)
 */
export function AddToolDialog({ tool, onAdd, open: externalOpen, onOpenChange: externalOnOpenChange }: AddToolDialogProps) {
    const t = useTranslations("tools.config");
    const { open, isEditMode, availableCategories, createTool, updateTool, form, watchCategoryNames, handleCategoryToggle, handleCategoryRemove, handleOpenChange, onSubmit, } = useAddToolDialogState({
        tool,
        onAdd,
        externalOpen,
        externalOnOpenChange,
        t,
    });
    const isSubmitting = createTool.isPending || updateTool.isPending;
    return (<Dialog open={open} onOpenChange={handleOpenChange}>
      {externalOpen === undefined && (<DialogTrigger render={<Button />}>
            <IconPlus className="h-5 w-5"/>
            {t("addTool")}
          </DialogTrigger>)}

      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-[700px]">
        <DialogHeader>
          <DialogTitle className="flex items-center space-x-2">
            <ToolIcon />
            <span>{isEditMode ? t("editTool") : t("addNewTool")}</span>
          </DialogTitle>
          <DialogDescription>
            {t("dialogDesc")}
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <div className="gap-6 grid py-4">
              <AddToolBasicInfoSection t={t} form={form} isPending={isSubmitting} availableCategories={availableCategories} selectedCategories={watchCategoryNames} onToggleCategory={handleCategoryToggle} onRemoveCategory={handleCategoryRemove}/>

              <AddToolCommandSection t={t} form={form} isPending={isSubmitting}/>
            </div>

            <AddToolDialogFooter t={t} isEditMode={isEditMode} isPending={isSubmitting} isFormValid={form.formState.isValid}/>
          </form>
        </Form>
      </DialogContent>
    </Dialog>);
}
