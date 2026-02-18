"use client"

import { AlertTriangle, FileText, Save, X } from "@/components/icons"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { CodeEditor } from "@/components/ui/code-editor"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface WordlistEditHeaderProps {
  t: TranslationFn
  name: string
}

export function WordlistEditHeader({ t, name }: WordlistEditHeaderProps) {
  return (
    <DialogHeader className="px-6 pt-6 pb-4 border-b">
      <DialogTitle className="flex items-center gap-2">
        <FileText className="h-5 w-5" />
        {t("title", { name })}
      </DialogTitle>
      <DialogDescription>{t("desc")}</DialogDescription>
    </DialogHeader>
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
      <div className="flex items-center gap-4 text-xs text-muted-foreground">
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
  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full border rounded-md">
        <div className="flex flex-col items-center gap-2">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
          <p className="text-sm text-muted-foreground">{t("loading")}</p>
        </div>
      </div>
    )
  }

  return (
    <CodeEditor
      value={content}
      onChange={onChange}
      language="plaintext"
      readOnly={readOnly}
      showLineNumbers
      showFoldGutter={false}
    />
  )
}

interface WordlistEditUnsavedNoticeProps {
  t: TranslationFn
}

export function WordlistEditUnsavedNotice({ t }: WordlistEditUnsavedNoticeProps) {
  return (
    <p className="flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400">
      <AlertTriangle className="h-3.5 w-3.5" />
      {t("unsavedChanges")}
    </p>
  )
}

interface WordlistEditFooterProps {
  t: TranslationFn
  isSaving: boolean
  hasChanges: boolean
  onCancel: () => void
  onSave: () => void
}

export function WordlistEditFooter({
  t,
  isSaving,
  hasChanges,
  onCancel,
  onSave,
}: WordlistEditFooterProps) {
  return (
    <DialogFooter className="px-6 py-4 border-t gap-2">
      <Button
        type="button"
        variant="outline"
        onClick={onCancel}
        disabled={isSaving}
      >
        <X className="h-4 w-4" />
        {t("cancel")}
      </Button>
      <Button
        type="button"
        onClick={onSave}
        disabled={isSaving || !hasChanges}
      >
        {isSaving ? (
          <>
            <div className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
            {t("saving")}
          </>
        ) : (
          <>
            <Save className="h-4 w-4" />
            {t("save")}
          </>
        )}
      </Button>
    </DialogFooter>
  )
}
