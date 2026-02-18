"use client"

import React from "react"
import { toast } from "sonner"
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
import {
  Dropzone,
  DropzoneContent,
  DropzoneEmptyState,
} from "@/components/ui/dropzone"
import { useImportFingerprintDialogState } from "@/components/fingerprints/import-fingerprint-dialog-state"
import type { FingerprintType } from "@/components/fingerprints/import-fingerprint-dialog-utils"

interface ImportFingerprintDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
  fingerprintType?: FingerprintType
  acceptedFileTypes?: string
}

export function ImportFingerprintDialog({
  open,
  onOpenChange,
  onSuccess,
  fingerprintType = "ehole",
  acceptedFileTypes,
}: ImportFingerprintDialogProps) {
  const t = useTranslations("tools.fingerprints")
  const tCommon = useTranslations("common.actions")
  const tToast = useTranslations("toast")
  const {
    files,
    config,
    importMutation,
    acceptConfig,
    handleDrop,
    handleImport,
    handleClose,
  } = useImportFingerprintDialogState({
    open,
    onOpenChange,
    onSuccess,
    fingerprintType,
    acceptedFileTypes,
    t,
    tToast,
  })

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
            maxSize={50 * 1024 * 1024}  // 50MB
            onError={(error) => toast.error(error.message)}
          >
            <DropzoneEmptyState />
            <DropzoneContent />
          </Dropzone>

          <p className="text-xs text-muted-foreground mt-3">
            {t("import.supportedFormat")}{" "}
            <code className="bg-muted px-1 rounded">
              {config.formatHint}
            </code>
          </p>
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
