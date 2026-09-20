"use client"

import { Upload, X, FileText } from "@/components/icons"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import * as React from "react"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { WordlistTagPicker } from "@/components/tools/wordlist-tag-picker"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface WordlistUploadHeaderProps {
  t: TranslationFn
}

export function WordlistUploadHeader({ t }: WordlistUploadHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle>{t("title")}</DialogTitle>
      <DialogDescription>{t("desc")}</DialogDescription>
    </DialogHeader>
  )
}

interface WordlistUploadDropzoneProps {
  t: TranslationFn
  file: File | null
  isDragActive: boolean
  onDragOver: (event: React.DragEvent) => void
  onDragLeave: (event: React.DragEvent) => void
  onDrop: (event: React.DragEvent) => void
  onFileSelect: (event: React.ChangeEvent<HTMLInputElement>) => void
  onRemoveFile: () => void
  formatFileSize: (bytes: number) => string
}

export function WordlistUploadDropzone({
  t,
  file,
  isDragActive,
  onDragOver,
  onDragLeave,
  onDrop,
  onFileSelect,
  onRemoveFile,
  formatFileSize,
}: WordlistUploadDropzoneProps) {
  const fileInputId = React.useId()

  return (
    <div
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
      className={cn(
        "relative flex flex-col items-center justify-center rounded-lg border-2 border-dashed transition-colors focus-within:border-ring focus-within:ring-2 focus-within:ring-ring/50",
        isDragActive
          ? "border-primary bg-primary/5"
          : "border-muted-foreground/25 hover:border-muted-foreground/50",
        file && "border-solid border-muted-foreground/25"
      )}
    >
      <input
        id={fileInputId}
        type="file"
        name="wordlistFile"
        accept=".txt"
        aria-label={t("selectFile")}
        className="sr-only"
        onChange={(event) => {
          onFileSelect(event)
          event.currentTarget.value = ""
        }}
      />
      {file ? (
        <div className="flex w-full items-center">
          <label
            htmlFor={fileInputId}
            className="flex min-w-0 flex-1 cursor-pointer items-center gap-3 p-4"
          >
            <div className="bg-primary/10 flex h-10 w-10 shrink-0 items-center justify-center rounded-lg">
              <FileText className="h-5 w-5 text-primary" />
            </div>
            <div className="min-w-0 flex-1">
              <p className={cn("truncate", textRole.bodyStrong)}>{file.name}</p>
              <p className={textRole.helperText}>
                {formatFileSize(file.size)}
              </p>
            </div>
          </label>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="mr-4 h-8 w-8 shrink-0"
            onClick={onRemoveFile}
            aria-label={t("removeFile")}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      ) : (
        <label
          htmlFor={fileInputId}
          className="flex w-full cursor-pointer flex-col items-center justify-center p-4"
        >
          <div className="bg-muted flex h-12 items-center justify-center rounded-full w-12">
            <Upload className="h-6 text-muted-foreground w-6" />
          </div>
          <div className="mt-3 text-center">
            <p className={textRole.bodyStrong}>{t("dragHint")}</p>
            <p className={cn("mt-1", textRole.helperText)}>
              {" "}
              <span className="text-primary">{t("selectFile")}</span>
            </p>
            <p className={cn("mt-2", textRole.helperText)}>
              {t("fileHint")}
            </p>
          </div>
        </label>
      )}
    </div>
  )
}

interface WordlistUploadFieldsProps {
  t: TranslationFn
  description: string
  tags: string[]
  onDescriptionChange: (value: string) => void
  onTagsChange: (value: string[]) => void
}

export function WordlistUploadFields({
  t,
  description,
  tags,
  onDescriptionChange,
  onTagsChange,
}: WordlistUploadFieldsProps) {
  return (
    <div className="gap-3 grid">
      <div className="space-y-2">
        <Label htmlFor="description">{t("descLabel")}</Label>
        <Input
          id="description"
          name="description"
          autoComplete="off"
          value={description}
          onChange={(event) => onDescriptionChange(event.target.value)}
          placeholder={t("descPlaceholder")}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="tags">{t("tagsLabel")}</Label>
        <WordlistTagPicker t={t} value={tags} onChange={onTagsChange} />
        <p className="text-muted-foreground text-xs">{t("tagsHint")}</p>
      </div>
    </div>
  )
}

interface WordlistUploadFooterProps {
  t: TranslationFn
  isPending: boolean
  canSubmit: boolean
}

export function WordlistUploadFooter({
  t,
  isPending,
  canSubmit,
}: WordlistUploadFooterProps) {
  return (
    <DialogFooter>
      <Button type="submit" disabled={isPending || !canSubmit}>
        {isPending ? t("uploading") : t("uploadButton")}
      </Button>
    </DialogFooter>
  )
}

interface WordlistUploadTriggerButtonProps {
  tWordlists: TranslationFn
  onClick?: () => void
  size?: React.ComponentProps<typeof Button>["size"]
}

export function WordlistUploadTriggerButton({ tWordlists, onClick, size }: WordlistUploadTriggerButtonProps) {
  return (
    <Button type="button" size={size} onClick={onClick}>
      <Upload className="h-4 mr-2 w-4" />
      {tWordlists("upload")}
    </Button>
  )
}
