"use client"

import * as React from "react"
import { AlertTriangle, FileText, Save } from "@/components/icons"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  LineNumberedTextarea,
  countLineNumberedTextareaLines,
  lineNumberedTextareaFillViewportClassName,
  lineNumberedTextareaResponsiveViewportClassName,
} from "@/components/common/line-numbered-textarea"
import { WordlistTagPicker } from "@/components/tools/wordlist-tag-picker"
import { cn } from "@/lib/utils"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface WordlistEditHeaderProps {
  t: TranslationFn
  name: string
}

export function WordlistEditHeader({ t, name }: WordlistEditHeaderProps) {
  return (
    <DialogHeader className="border-b pb-4 pt-6 px-6">
      <DialogTitle className="flex gap-2 items-center">
        <FileText className="h-5 w-5" />
        {t("title", { name })}
      </DialogTitle>
      <DialogDescription>{t("desc")}</DialogDescription>
    </DialogHeader>
  )
}

interface WordlistEditMetadataProps {
  t: TranslationFn
  description: string
  tags: string[]
  onDescriptionChange: (value: string) => void
  onTagsChange: (tags: string[]) => void
}

export function WordlistEditMetadata({
  t,
  description,
  tags,
  onDescriptionChange,
  onTagsChange,
}: WordlistEditMetadataProps) {
  return (
    <section className="space-y-3">
      <div className="space-y-2">
        <Label htmlFor="wordlist-description">{t("description")}</Label>
        <Input
          id="wordlist-description"
          value={description}
          onChange={(event) => onDescriptionChange(event.target.value)}
        />
      </div>
      <div className="space-y-2">
        <Label>{t("tags")}</Label>
        <WordlistTagPicker t={t} value={tags} onChange={onTagsChange} />
      </div>
    </section>
  )
}

interface WordlistEditMetaProps {
  t: TranslationFn
  lineCount: number
  fileHashShort: string | null
  fileHashFull?: string | null
}

export function WordlistEditMeta({
  t,
  lineCount,
  fileHashShort,
  fileHashFull,
}: WordlistEditMetaProps) {
  return (
    <div className="flex items-center justify-between">
      <Label>{t("content")}</Label>
      <div className="flex gap-4 items-center text-muted-foreground text-xs">
        <span>{t("lines", { count: lineCount.toLocaleString() })}</span>
        {fileHashShort && (
          <span title={fileHashFull || fileHashShort}>
            {t("hash")}: {fileHashShort}
          </span>
        )}
      </div>
    </div>
  )
}

interface WordlistEditEditorProps {
  t: TranslationFn
  isLoading: boolean
  content: string
  onChange: (value: string) => void
  readOnly: boolean
}

export function WordlistEditEditor({
  t,
  isLoading,
  content,
  onChange,
  readOnly,
}: WordlistEditEditorProps) {
  const lineNumbersRef = React.useRef<HTMLDivElement | null>(null)
  const textareaRef = React.useRef<HTMLTextAreaElement | null>(null)

  if (isLoading) {
    return (
      <div aria-busy="true" aria-label={t("loading")} role="status" className="min-h-0 flex-1 border rounded-md">
        <div className="h-full" />
      </div>
    )
  }

  return (
    <LineNumberedTextarea
      id="wordlist-content"
      name="wordlist-content"
      aria-label={t("content")}
      lineCount={Math.max(countLineNumberedTextareaLines(content), 8)}
      lineNumbersRef={lineNumbersRef}
      textareaRef={textareaRef}
      viewportClassName={cn(lineNumberedTextareaResponsiveViewportClassName, lineNumberedTextareaFillViewportClassName)}
      value={content}
      onChange={(event) => onChange(event.target.value)}
      readOnly={readOnly}
    />
  )
}

interface WordlistEditUnsavedNoticeProps {
  t: TranslationFn
}

export function WordlistEditUnsavedNotice({ t }: WordlistEditUnsavedNoticeProps) {
  return (
    <p className="flex gap-1 items-center text-warning text-xs">
      <AlertTriangle className="h-3.5 w-3.5" />
      {t("unsavedChanges")}
    </p>
  )
}

interface WordlistEditFooterProps {
  t: TranslationFn
  isSaving: boolean
  isSavingMetadata: boolean
  hasChanges: boolean
  hasMetadataChanges: boolean
  onSaveDialog: () => void
  canEditContent: boolean
}

export function WordlistEditFooter({
  t,
  isSaving,
  isSavingMetadata,
  hasChanges,
  hasMetadataChanges,
  onSaveDialog,
  canEditContent,
}: WordlistEditFooterProps) {
  const isSavingAny = isSaving || isSavingMetadata
  const hasAnyChanges = hasChanges || hasMetadataChanges
  const canSave = (canEditContent && hasChanges) || hasMetadataChanges

  return (
    <DialogFooter className="border-t gap-2 px-6 py-4">
      {canEditContent || hasAnyChanges ? (
        <Button
          type="button"
          onClick={onSaveDialog}
          disabled={isSavingAny || !canSave}
          loading={isSavingAny}
          loadingLabel={t("saving")}
        >
          <Save className="h-4 w-4" />
          {t("save")}
        </Button>
      ) : null}
    </DialogFooter>
  )
}
