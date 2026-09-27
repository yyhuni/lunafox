"use client";
import React from "react";
import { semanticIcons } from "@/components/icons";
import { IconPlus } from "@/components/icons";
import { IconX } from "@/components/icons";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { CategoryNameMap } from "@/types/tool.types";
import { compactSurfaceClassNames } from "@/lib/ui/compact-surface-contract";
import { cn } from "@/lib/utils";
import { textRole } from "@/lib/typography";
type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string;
const ToolIcon = semanticIcons.concept.tool;
interface AddCustomToolDialogHeaderProps {
    t: TranslationFn;
    isEditMode: boolean;
}
export function AddCustomToolDialogHeader({ t, isEditMode }: AddCustomToolDialogHeaderProps) {
    return (<DialogHeader>
      <DialogTitle className="flex items-center space-x-2">
        <ToolIcon />
        <span>{isEditMode ? t("editCustomTool") : t("addCustomTool")}</span>
      </DialogTitle>
      <DialogDescription>{t("customDialogDesc")}</DialogDescription>
    </DialogHeader>);
}
interface AddCustomToolNameFieldProps {
    t: TranslationFn;
    value: string;
    isSubmitting: boolean;
    onChange: (value: string) => void;
}
export function AddCustomToolNameField({ t, value, isSubmitting, onChange, }: AddCustomToolNameFieldProps) {
    return (<div className="gap-2 grid">
      <Label htmlFor="name">
        {t("toolName")} <span className="text-error">*</span>
      </Label>
      <Input id="name" name="name" size="sm" autoComplete="off" placeholder={t("customToolNamePlaceholder")} value={value} onChange={(event) => onChange(event.target.value)} disabled={isSubmitting} required/>
    </div>);
}
interface AddCustomToolDescriptionFieldProps {
    t: TranslationFn;
    value: string;
    isSubmitting: boolean;
    onChange: (value: string) => void;
}
export function AddCustomToolDescriptionField({ t, value, isSubmitting, onChange, }: AddCustomToolDescriptionFieldProps) {
    return (<div className="gap-2 grid">
      <Label htmlFor="description">{t("toolDesc")}</Label>
      <Textarea id="description" name="description" autoComplete="off" placeholder={t("customToolDescPlaceholder")} value={value} onChange={(event) => onChange(event.target.value)} disabled={isSubmitting} rows={3}/>
    </div>);
}
interface AddCustomToolPathFieldProps {
    t: TranslationFn;
    value: string;
    isSubmitting: boolean;
    onChange: (value: string) => void;
}
export function AddCustomToolPathField({ t, value, isSubmitting, onChange, }: AddCustomToolPathFieldProps) {
    return (<div className="gap-2 grid">
      <Label htmlFor="directory">
        {t("toolPath")} <span className="text-error">*</span>
      </Label>
      <Input id="directory" name="directory" size="sm" autoComplete="off" placeholder={t("toolPathPlaceholder")} value={value} onChange={(event) => onChange(event.target.value)} disabled={isSubmitting} required/>
      <p className="text-muted-foreground text-xs">{t("toolPathHint")}</p>
    </div>);
}
interface AddCustomToolCategoriesFieldProps {
    t: TranslationFn;
    isSubmitting: boolean;
    availableCategories: string[];
    selectedCategories: string[];
    onToggle: (categoryName: string) => void;
    onRemove: (categoryName: string) => void;
}
export function AddCustomToolCategoriesField({ t, isSubmitting, availableCategories, selectedCategories, onToggle, onRemove, }: AddCustomToolCategoriesFieldProps) {
    return (<div className="gap-2 grid">
      <Label>{t("categoryTags")}</Label>

      {selectedCategories.length > 0 && (<div className={cn(compactSurfaceClassNames.mutedInfo, "flex flex-wrap gap-2")}>
          {selectedCategories.map((categoryName) => (<Badge key={categoryName} size="tag" variant="default" className="flex gap-1 items-center">
              {CategoryNameMap[categoryName] || categoryName}
              <button type="button" onClick={() => onRemove(categoryName)} disabled={isSubmitting} className="hover:bg-primary/20 ml-1 p-0.5 rounded-full">
                <IconX className="h-3 w-3"/>
              </button>
            </Badge>))}
        </div>)}

      <div className={cn(compactSurfaceClassNames.info, "flex flex-wrap gap-2")}>
        {availableCategories.length > 0 ? (availableCategories.map((categoryName) => {
            const isSelected = selectedCategories.includes(categoryName);
            return (<Badge key={categoryName} variant={isSelected ? "secondary" : "outline"} className="hover:bg-secondary/80 transition-colors" render={<button type="button" onClick={() => onToggle(categoryName)}/>}>
                  {CategoryNameMap[categoryName] || categoryName}
                </Badge>);
        })) : (<p className={textRole.helperText}>{t("noCategories")}</p>)}
      </div>
    </div>);
}
interface AddCustomToolFooterProps {
    t: TranslationFn;
    isEditMode: boolean;
    isSubmitting: boolean;
    isFormValid: boolean;
}
export function AddCustomToolFooter({ t, isEditMode, isSubmitting, isFormValid, }: AddCustomToolFooterProps) {
    return (<DialogFooter>
      <Button type="submit" disabled={isSubmitting || !isFormValid} loading={isSubmitting} loadingLabel={isEditMode ? t("saving") : t("creating")}>
        <>
          <IconPlus className="h-5 w-5"/>
          {isEditMode ? t("saveChanges") : t("createTool")}
        </>
      </Button>
    </DialogFooter>);
}
