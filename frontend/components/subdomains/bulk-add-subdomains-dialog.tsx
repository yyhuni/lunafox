"use client";
import React from "react";
import { Plus } from "@/components/icons";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTrigger } from "@/components/ui/dialog";
import { scrollableFormDialogContentClassName } from "@/lib/ui/overlay-styles";
import { useBulkAddSubdomainsDialogState } from "@/components/subdomains/bulk-add-subdomains-dialog-state";
import { BulkAddSubdomainsFooter, BulkAddSubdomainsHeader, BulkAddSubdomainsInput, } from "@/components/subdomains/bulk-add-subdomains-dialog-sections";
interface BulkAddSubdomainsDialogProps {
    targetId: number;
    targetName?: string;
    open?: boolean;
    onOpenChange?: (open: boolean) => void;
    onSuccess?: () => void;
}
/**
 * Bulk add subdomains dialog component
 *
 * Following the design pattern of AddTargetDialog, provides a text input with line numbers,
 * supporting real-time validation and error prompts.
 */
export function BulkAddSubdomainsDialog({ targetId, targetName, open: externalOpen, onOpenChange: externalOnOpenChange, onSuccess, }: BulkAddSubdomainsDialogProps) {
    const t = useTranslations("bulkAdd.subdomain");
    const { open, handleOpenChange, inputText, validationResult, lineNumbersRef, textareaRef, bulkCreateSubdomains, handleInputChange, handleSubmit, handleTextareaScroll, lineCount, isFormValid, } = useBulkAddSubdomainsDialogState({
        targetId,
        open: externalOpen,
        onOpenChange: externalOnOpenChange,
        onSuccess,
    });
    return (<Dialog open={open} onOpenChange={handleOpenChange}>
      {externalOpen === undefined && (<DialogTrigger render={<Button size="sm" variant="outline"/>}>
            <Plus className="h-4 w-4"/>
            {t("bulkAdd")}
          </DialogTrigger>)}

      <DialogContent className={scrollableFormDialogContentClassName}>
        <BulkAddSubdomainsHeader title={t("title")} description={t("description")} targetName={targetName} t={t}/>

        <form onSubmit={handleSubmit}>
          <div className="gap-4 grid py-4">
            <BulkAddSubdomainsInput t={t} inputText={inputText} placeholder={t("placeholder")} lineNumbersRef={lineNumbersRef} textareaRef={textareaRef} onInputChange={handleInputChange} onScroll={handleTextareaScroll} isPending={bulkCreateSubdomains.isPending} lineCount={lineCount} validationResult={validationResult}/>
          </div>

          <BulkAddSubdomainsFooter t={t} isPending={bulkCreateSubdomains.isPending} isFormValid={isFormValid}/>
        </form>
      </DialogContent>
    </Dialog>);
}
