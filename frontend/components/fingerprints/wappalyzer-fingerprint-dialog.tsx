"use client"

import React from "react"
import { useTranslations } from "next-intl"
import {
  Dialog,
  DialogContent,
} from "@/components/ui/dialog"
import {
  WappalyzerFingerprintDialogHeader,
  WappalyzerFingerprintBasicFields,
  WappalyzerFingerprintDetectionFields,
  WappalyzerFingerprintDialogFooter,
} from "@/components/fingerprints/wappalyzer-fingerprint-dialog-sections"
import { useWappalyzerFingerprintDialogState } from "@/components/fingerprints/wappalyzer-fingerprint-dialog-state"
import type { WappalyzerFingerprint } from "@/types/fingerprint.types"

interface WappalyzerFingerprintDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  fingerprint?: WappalyzerFingerprint | null
  onSuccess?: () => void
}

export function WappalyzerFingerprintDialog({
  open,
  onOpenChange,
  fingerprint,
  onSuccess,
}: WappalyzerFingerprintDialogProps) {
  const t = useTranslations("tools.fingerprints")
  const tCommon = useTranslations("common.actions")

  const {
    isEdit,
    register,
    handleSubmit,
    formState,
    onSubmit,
  } = useWappalyzerFingerprintDialogState({
    fingerprint,
    onOpenChange,
    onSuccess,
    t,
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[80vh] overflow-y-auto">
        <WappalyzerFingerprintDialogHeader t={t} isEdit={isEdit} />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <WappalyzerFingerprintBasicFields
            t={t}
            tCommon={tCommon}
            register={register}
            errors={formState.errors}
          />
          <WappalyzerFingerprintDetectionFields
            t={t}
            register={register}
          />
          <WappalyzerFingerprintDialogFooter
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
