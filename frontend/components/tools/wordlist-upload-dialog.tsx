"use client";
import type { ComponentProps, ReactElement } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTrigger, } from "@/components/ui/dialog";
import { useWordlistUploadDialogState } from "@/components/tools/wordlist-upload-dialog-state";
import { WordlistUploadHeader, WordlistUploadDropzone, WordlistUploadFields, WordlistUploadFooter, WordlistUploadTriggerButton, } from "@/components/tools/wordlist-upload-dialog-sections";
interface WordlistUploadDialogProps {
    trigger?: ReactElement;
    triggerSize?: ComponentProps<typeof Button>["size"];
}
export function WordlistUploadDialog({ trigger, triggerSize }: WordlistUploadDialogProps) {
    const t = useTranslations("tools.wordlists.uploadDialog");
    const tWordlists = useTranslations("tools.wordlists");
    const { open, description, tags, file, isDragActive, isPending, setDescription, setTags, handleOpenChange, handleDragOver, handleDragLeave, handleDrop, handleFileSelect, handleSubmit, removeFile, formatFileSize, } = useWordlistUploadDialogState();
    return (<Dialog open={open} onOpenChange={handleOpenChange}>
      {trigger ? (<DialogTrigger render={trigger}></DialogTrigger>) : (<WordlistUploadTriggerButton tWordlists={tWordlists} size={triggerSize} onClick={() => handleOpenChange(true)}/>)}
      <DialogContent className="sm:max-w-lg">
        <WordlistUploadHeader t={t}/>

        <form onSubmit={handleSubmit} className="space-y-4">
          <WordlistUploadDropzone t={t} file={file} isDragActive={isDragActive} onDragOver={handleDragOver} onDragLeave={handleDragLeave} onDrop={handleDrop} onFileSelect={handleFileSelect} onRemoveFile={removeFile} formatFileSize={formatFileSize}/>

          <WordlistUploadFields t={t} description={description} tags={tags} onDescriptionChange={setDescription} onTagsChange={setTags}/>

          <WordlistUploadFooter t={t} isPending={isPending} canSubmit={!!file}/>
        </form>
      </DialogContent>
    </Dialog>);
}
