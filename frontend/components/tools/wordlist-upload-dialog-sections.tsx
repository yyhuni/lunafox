"use client"

import { Upload, X, FileText } from "@/components/icons"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
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
        <div className="flex w-full items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
            <FileText className="h-5 w-5 text-primary" />
          </div>
          <div className="flex-1 min-w-0">
            <p className="truncate font-medium text-sm">{file.name}</p>
            <p className="text-xs text-muted-foreground">
              {formatFileSize(file.size)}
            </p>
          </div>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-8 w-8 shrink-0"
            onClick={onRemoveFile}
            aria-label={t("removeFile")}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      ) : (
        <>
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted">
            <Upload className="h-6 w-6 text-muted-foreground" />
          </div>
          <div className="mt-3 text-center">
            <p className="text-sm font-medium">{t("dragHint")}</p>
            <p className="mt-1 text-xs text-muted-foreground">
              {" "}
              <label className="cursor-pointer text-primary hover:underline">
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
            <p className="mt-2 text-xs text-muted-foreground">
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
  tWordlists: TranslationFn
  name: string
  description: string
  onNameChange: (value: string) => void
  onDescriptionChange: (value: string) => void
}

export function WordlistUploadFields({
  t,
  tWordlists,
  name,
  description,
  onNameChange,
  onDescriptionChange,
}: WordlistUploadFieldsProps) {
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <div className="space-y-2">
        <Label htmlFor="name">
          {tWordlists("name")} <span className="text-destructive">*</span>
        </Label>
        <Input
          id="name"
          name="name"
          autoComplete="off"
          value={name}
          onChange={(event) => onNameChange(event.target.value)}
          placeholder={t("namePlaceholder")}
        />
      </div>
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
    </div>
  )
}

interface WordlistUploadFooterProps {
  t: TranslationFn
  isPending: boolean
  canSubmit: boolean
  onCancel: () => void
}

export function WordlistUploadFooter({
  t,
  isPending,
  canSubmit,
  onCancel,
}: WordlistUploadFooterProps) {
  return (
    <DialogFooter>
      <Button
        type="button"
        variant="outline"
        onClick={onCancel}
        disabled={isPending}
      >
        {t("cancel")}
      </Button>
      <Button type="submit" disabled={isPending || !canSubmit}>
        {isPending ? t("uploading") : t("uploadButton")}
      </Button>
    </DialogFooter>
  )
}

interface WordlistUploadTriggerButtonProps {
  tWordlists: TranslationFn
}

export function WordlistUploadTriggerButton({ tWordlists }: WordlistUploadTriggerButtonProps) {
  return (
    <Button>
      <Upload className="mr-2 h-4 w-4" />
      {tWordlists("upload")}
    </Button>
  )
}
