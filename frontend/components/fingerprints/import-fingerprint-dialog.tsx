"use client"

import React from "react"
import { toastFeedback } from "@/lib/toast-helpers"
import { useTranslations } from "next-intl"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { IconAlertTriangle } from "@/components/icons"
import {
  Dropzone,
  DropzoneContent,
  DropzoneEmptyState,
} from "@/components/shared/upload/dropzone"
import {
  getFingerprintImportReasonTranslationKey,
} from "@/components/fingerprints/fingerprint-import-diagnostic"
import { useImportFingerprintDialogState } from "@/components/fingerprints/import-fingerprint-dialog-state"
import { FINGERPRINT_IMPORT_MAX_FILE_SIZE } from "@/components/fingerprints/import-fingerprint-dialog-utils"

interface ImportFingerprintDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

export function ImportFingerprintDialog({
  open,
  onOpenChange,
  onSuccess,
}: ImportFingerprintDialogProps) {
  const t = useTranslations("tools.fingerprints")
  const tCommon = useTranslations("common.actions")
  const tToast = useTranslations("toast")
  const {
    files,
    config,
    importFailure,
    importMutation,
    acceptConfig,
    handleDrop,
    handleImport,
    handleClose,
  } = useImportFingerprintDialogState({
    open,
    onOpenChange,
    onSuccess,
    t,
    tToast,
  })

  const diagnostic = importFailure?.diagnostic
  const reasonKey = getFingerprintImportReasonTranslationKey(importFailure?.reason ?? null)
  const reason = importFailure
    ? reasonKey
      ? t(reasonKey)
      : t("import.diagnostic.unknownReason")
    : null
  const recordLocation = diagnostic?.kind === "RECORD" && diagnostic.recordIndex !== undefined
    ? diagnostic.fieldPath
      ? t("import.diagnostic.recordLocation", {
          recordIndex: diagnostic.recordIndex,
          fieldPath: diagnostic.fieldPath,
        })
      : t("import.diagnostic.recordIndex", { recordIndex: diagnostic.recordIndex })
    : null
  const parserLocation =
    diagnostic?.kind === "SYNTAX" || diagnostic?.kind === "ENCODING"
      ? diagnostic.line !== undefined && diagnostic.column !== undefined
        ? t("import.diagnostic.parserLocation", {
            line: diagnostic.line,
            column: diagnostic.column,
          })
        : diagnostic.line !== undefined
          ? t("import.diagnostic.parserLine", { line: diagnostic.line })
          : diagnostic.column !== undefined
            ? t("import.diagnostic.parserColumn", { column: diagnostic.column })
            : null
      : null

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>{config.title}</DialogTitle>
          <DialogDescription>
            {config.description}
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          <Dropzone
            src={files}
            onDrop={handleDrop}
            accept={acceptConfig}
            maxFiles={1}
            multiple={false}
            maxSize={FINGERPRINT_IMPORT_MAX_FILE_SIZE}
            onError={() => toastFeedback.error(t("import.fileRejected"))}
          >
            <DropzoneEmptyState />
            <DropzoneContent />
          </Dropzone>

          <p className="mt-3 text-muted-foreground text-xs">
            {t("import.supportedFormat")}{" "}
            <code className="bg-muted px-1 rounded">
              {config.formatHint}
            </code>
          </p>

          {importFailure ? (
            <Alert variant="destructive" className="mt-4">
              <IconAlertTriangle aria-hidden="true" />
              <AlertTitle>{t("import.diagnostic.title")}</AlertTitle>
              <AlertDescription>
                <p>{reason}</p>
                {recordLocation ? <p>{recordLocation}</p> : null}
                {parserLocation ? <p>{parserLocation}</p> : null}
              </AlertDescription>
            </Alert>
          ) : null}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => handleClose(false)}>
            {tCommon("cancel")}
          </Button>
          <Button
            onClick={handleImport}
            disabled={files.length === 0 || importMutation.isPending}
          >
            {importMutation.isPending ? t("import.importing") : tCommon("import")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
