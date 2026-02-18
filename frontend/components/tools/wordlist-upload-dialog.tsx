"use client"

import type { ReactNode } from "react"
import { useTranslations } from "next-intl"
import {
  Dialog,
  DialogContent,
  DialogTrigger,
} from "@/components/ui/dialog"
import { useWordlistUploadDialogState } from "@/components/tools/wordlist-upload-dialog-state"
import {
  WordlistUploadHeader,
  WordlistUploadDropzone,
  WordlistUploadFields,
  WordlistUploadFooter,
  WordlistUploadTriggerButton,
} from "@/components/tools/wordlist-upload-dialog-sections"

interface WordlistUploadDialogProps {
  trigger?: ReactNode
}

export function WordlistUploadDialog({ trigger }: WordlistUploadDialogProps) {
  const t = useTranslations("tools.wordlists.uploadDialog")
  const tWordlists = useTranslations("tools.wordlists")

  const {
    open,
    name,
    description,
    file,
    isDragActive,
    isPending,
    setName,
    setDescription,
    handleOpenChange,
    handleDragOver,
    handleDragLeave,
    handleDrop,
    handleFileSelect,
    handleSubmit,
    removeFile,
    formatFileSize,
  } = useWordlistUploadDialogState()

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        {trigger || <WordlistUploadTriggerButton tWordlists={tWordlists} />}
      </DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <WordlistUploadHeader t={t} />

        <form onSubmit={handleSubmit} className="space-y-4">
          <WordlistUploadDropzone
            t={t}
            file={file}
            isDragActive={isDragActive}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
            onFileSelect={handleFileSelect}
            onRemoveFile={removeFile}
            formatFileSize={formatFileSize}
          />

          <WordlistUploadFields
            t={t}
            tWordlists={tWordlists}
            name={name}
            description={description}
            onNameChange={setName}
            onDescriptionChange={setDescription}
          />

          <WordlistUploadFooter
            t={t}
            isPending={isPending}
            canSubmit={!!file && !!name}
            onCancel={() => handleOpenChange(false)}
          />
        </form>
      </DialogContent>
    </Dialog>
  )
}
