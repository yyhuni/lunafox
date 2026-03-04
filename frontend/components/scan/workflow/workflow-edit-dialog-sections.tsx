"use client"

import { AlertCircle, AlertTriangle, CheckCircle2, FileCode, Save, X } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { CodeEditor } from "@/components/ui/code-editor"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import type { WorkflowEditYamlError } from "@/components/scan/workflow/workflow-edit-dialog-state"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface WorkflowEditHeaderProps {
  workflowName?: string
  t: TranslationFn
}

export function WorkflowEditHeader({ workflowName, t }: WorkflowEditHeaderProps) {
  return (
    <DialogHeader className="px-6 pt-6 pb-4 border-b">
      <DialogTitle className="flex items-center gap-2">
        <FileCode className="h-5 w-5" />
        {t("title", { name: workflowName ?? "" })}
      </DialogTitle>
      <DialogDescription>{t("desc")}</DialogDescription>
    </DialogHeader>
  )
}

interface WorkflowEditEditorProps {
  t: TranslationFn
  tToast: TranslationFn
  yamlContent: string
  yamlError: WorkflowEditYamlError
  isSubmitting: boolean
  onChange: (value: string) => void
}

export function WorkflowEditEditor({
  t,
  tToast,
  yamlContent,
  yamlError,
  isSubmitting,
  onChange,
}: WorkflowEditEditorProps) {
  return (
    <div className="flex-1 overflow-hidden px-6 py-4">
      <div className="flex flex-col h-full gap-2">
        <div className="flex items-center justify-between">
          <Label>{t("yamlConfig")}</Label>
          <div className="flex items-center gap-2">
            {yamlContent.trim() && (
              yamlError ? (
                <div className="flex items-center gap-1 text-xs text-destructive">
                  <AlertCircle className="h-3.5 w-3.5" />
                  <span>{t("syntaxError")}</span>
                </div>
              ) : (
                <div className="flex items-center gap-1 text-xs text-green-600 dark:text-green-400">
                  <CheckCircle2 className="h-3.5 w-3.5" />
                  <span>{t("syntaxValid")}</span>
                </div>
              )
            )}
          </div>
        </div>

        <CodeEditor
          value={yamlContent}
          onChange={onChange}
          language="yaml"
          readOnly={isSubmitting}
          className={yamlError ? "border-destructive" : ""}
          showLineNumbers
          showFoldGutter
        />

        {yamlError && (
          <div className="flex items-start gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-md">
            <AlertCircle className="h-4 w-4 text-destructive mt-0.5 flex-shrink-0" />
            <div className="flex-1 text-xs">
              <p className="font-semibold text-destructive mb-1">
                {yamlError.line && yamlError.column
                  ? t("errorLocation", { line: yamlError.line, column: yamlError.column })
                  : tToast("yamlSyntaxError")}
              </p>
              <p className="text-muted-foreground">{yamlError.message}</p>
            </div>
          </div>
        )}
        <p className="flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400">
          <AlertTriangle className="h-3.5 w-3.5" />
          {t("unsavedChanges")}
        </p>
      </div>
    </div>
  )
}

interface WorkflowEditFooterProps {
  t: TranslationFn
  tCommon: TranslationFn
  isSubmitting: boolean
  hasChanges: boolean
  yamlError: WorkflowEditYamlError
  onCancel: () => void
  onSave: () => void
}

export function WorkflowEditFooter({
  t,
  tCommon,
  isSubmitting,
  hasChanges,
  yamlError,
  onCancel,
  onSave,
}: WorkflowEditFooterProps) {
  return (
    <DialogFooter className="px-6 py-4 border-t gap-2">
      <Button
        type="button"
        variant="outline"
        onClick={onCancel}
        disabled={isSubmitting}
      >
        <X className="h-4 w-4" />
        {tCommon("cancel")}
      </Button>
      <Button
        type="button"
        onClick={onSave}
        disabled={isSubmitting || !hasChanges || !!yamlError}
      >
        {isSubmitting ? (
          <>
            <div className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
            {t("saving")}
          </>
        ) : (
          <>
            <Save className="h-4 w-4" />
            {t("saveConfig")}
          </>
        )}
      </Button>
    </DialogFooter>
  )
}
