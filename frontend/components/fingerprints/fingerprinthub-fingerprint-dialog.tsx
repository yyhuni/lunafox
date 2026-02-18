"use client"

import React from "react"
import { useTranslations } from "next-intl"
import { Dialog, DialogContent } from "@/components/ui/dialog"
import { useFingerPrintHubFingerprintDialogState } from "@/components/fingerprints/fingerprinthub-fingerprint-dialog-state"
import {
  FingerPrintHubDialogHeader,
  FingerPrintHubIdNameFields,
  FingerPrintHubAuthorSeverityFields,
  FingerPrintHubTagsField,
  FingerPrintHubMetadataField,
  FingerPrintHubHttpField,
  FingerPrintHubSourceFileField,
  FingerPrintHubDialogFooter,
} from "@/components/fingerprints/fingerprinthub-fingerprint-dialog-sections"
import type { FingerPrintHubFingerprint } from "@/types/fingerprint.types"

interface FingerPrintHubFingerprintDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  fingerprint?: FingerPrintHubFingerprint | null
  onSuccess?: () => void
}

export function FingerPrintHubFingerprintDialog({
  open,
  onOpenChange,
  fingerprint,
  onSuccess,
}: FingerPrintHubFingerprintDialogProps) {
  const t = useTranslations("tools.fingerprints")
  const tCommon = useTranslations("common.actions")

  const {
    isEdit,
    register,
    handleSubmit,
    setValue,
    watch,
    formState,
    onSubmit,
  } = useFingerPrintHubFingerprintDialogState({
    fingerprint,
    onOpenChange,
    onSuccess,
    t,
  })

  const severity = watch("severity")

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[700px] max-h-[85vh] overflow-y-auto">
        <FingerPrintHubDialogHeader t={t} isEdit={isEdit} />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <FingerPrintHubIdNameFields
            t={t}
            register={register}
            errors={formState.errors}
          />

          <FingerPrintHubAuthorSeverityFields
            t={t}
            severity={severity}
            onSeverityChange={(value) => setValue("severity", value)}
            register={register}
          />

          <FingerPrintHubTagsField
            t={t}
            register={register}
          />

          <FingerPrintHubMetadataField
            t={t}
            register={register}
          />

          <FingerPrintHubHttpField
            t={t}
            register={register}
            errors={formState.errors}
          />

          <FingerPrintHubSourceFileField
            t={t}
            register={register}
          />

          <FingerPrintHubDialogFooter
            tCommon={tCommon}
            isEdit={isEdit}
            isSubmitting={formState.isSubmitting}
            onCancel={() => onOpenChange(false)}
          />
        </form>
      </DialogContent>
    </Dialog>
  )
}
