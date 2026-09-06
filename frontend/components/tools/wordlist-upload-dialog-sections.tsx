"use client"

import { Upload, X, FileText } from "@/components/icons"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import type * as React from "react"
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
  return (
    <div
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
      className={cn(
        "relative flex flex-col items-center justify-center rounded-lg border-2 border-dashed p-6 transition-colors",
        isDragActive
          ? "border-primary bg-primary/5"
          : "border-muted-foreground/25 hover:border-muted-foreground/50",
        file && "border-solid border-muted-foreground/25"
      )}
    >
      {file ? (
        <div className="flex gap-3 items-center w-full">
          <div className="bg-primary/10 flex h-10 items-center justify-center rounded-lg w-10">
            <FileText className="h-5 text-primary w-5" />
          </div>
          <div className="flex-1 min-w-0">
            <p className={cn("truncate", textRole.bodyStrong)}>{file.name}</p>
            <p className={textRole.helperText}>
              {formatFileSize(file.size)}
            </p>
          </div>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-8 shrink-0 w-8"
            onClick={onRemoveFile}
            aria-label={t("removeFile")}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      ) : (
        <>
          <div className="bg-muted flex h-12 items-center justify-center rounded-full w-12">
            <Upload className="h-6 text-muted-foreground w-6" />
          </div>
          <div className="mt-3 text-center">
            <p className={textRole.bodyStrong}>{t("dragHint")}</p>
            <p className={cn("mt-1", textRole.helperText)}>
              {" "}
              <label className="cursor-pointer hover:underline text-primary">
                {t("selectFile")}
                <input
                  type="file"
                  name="wordlistFile"
                  accept=".txt"
                  aria-label={t("selectFile")}
                  className="hidden"
                  onChange={onFileSelect}
                />
              </label>
            </p>
            <p className={cn("mt-2", textRole.helperText)}>
              {t("fileHint")}
            </p>
          </div>
        </>
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
    <div className="gap-4 grid">
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
