"use client"

import { AlertCircle, AlertTriangle, CheckCircle2, FileCode, Save } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { CodeEditor } from "@/components/shared/editors/code-editor"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import type { WorkflowEditYamlError } from "@/components/scan/workflow/workflow-edit-dialog-state"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface WorkflowEditHeaderProps {
  workflowName?: string
  t: TranslationFn
}

export function WorkflowEditHeader({ workflowName, t }: WorkflowEditHeaderProps) {
  return (
    <DialogHeader className="border-b pb-4 pt-6 px-6">
      <DialogTitle className="flex gap-2 items-center">
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
      <div className="flex flex-col gap-2 h-full">
        <div className="flex items-center justify-between">
          <Label>{t("yamlConfig")}</Label>
          <div className="flex gap-2 items-center">
            {yamlContent.trim() && (
              yamlError ? (
                <div className="flex gap-1 items-center text-destructive text-xs">
                  <AlertCircle className="h-3.5 w-3.5" />
                  <span>{t("syntaxError")}</span>
                </div>
              ) : (
                <div className="flex gap-1 items-center text-success text-xs">
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
          <div className="bg-destructive/10 border border-destructive/20 flex gap-2 items-start p-3 rounded-md">
            <AlertCircle className="flex-shrink-0 h-4 mt-0.5 text-destructive w-4" />
            <div className="flex-1 text-xs">
              <p className={cn(textRole.bodyStrong, "mb-1 text-destructive")}>
                {yamlError.line && yamlError.column
                  ? t("errorLocation", { line: yamlError.line, column: yamlError.column })
                  : tToast("yamlSyntaxError")}
              </p>
              <p className="text-muted-foreground">{yamlError.message}</p>
            </div>
          </div>
        )}
        <p className="flex gap-1 items-center text-warning text-xs">
          <AlertTriangle className="h-3.5 w-3.5" />
          {t("unsavedChanges")}
        </p>
      </div>
    </div>
  )
}

interface WorkflowEditFooterProps {
  t: TranslationFn
  isSubmitting: boolean
  hasChanges: boolean
  yamlError: WorkflowEditYamlError
  onSave: () => void
}

export function WorkflowEditFooter({
  t,
  isSubmitting,
  hasChanges,
  yamlError,
  onSave,
}: WorkflowEditFooterProps) {
  return (
    <DialogFooter className="border-t gap-2 px-6 py-4">
      <Button
        type="button"
        onClick={onSave}
        disabled={isSubmitting || !hasChanges || !!yamlError}
        loading={isSubmitting}
        loadingLabel={t("saving")}
      >
        <Save className="h-4 w-4" />
        {t("saveConfig")}
      </Button>
    </DialogFooter>
  )
}
