"use client"

import React from "react"
import { useTranslations } from "next-intl"
import { Dialog, DialogContent } from "@/components/ui/dialog"
import { useARLFingerprintDialogState } from "@/components/fingerprints/arl-fingerprint-dialog-state"
import {
  ARLFingerprintDialogHeader,
  ARLFingerprintNameField,
  ARLFingerprintRuleField,
  ARLFingerprintDialogFooter,
} from "@/components/fingerprints/arl-fingerprint-dialog-sections"
import type { ARLFingerprint } from "@/types/fingerprint.types"

interface ARLFingerprintDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  fingerprint?: ARLFingerprint | null
  onSuccess?: () => void
}

export function ARLFingerprintDialog({
  open,
  onOpenChange,
  fingerprint,
  onSuccess,
}: ARLFingerprintDialogProps) {
  const t = useTranslations("tools.fingerprints")
  const tCommon = useTranslations("common.actions")

  const {
    isEdit,
    register,
    handleSubmit,
    formState,
    onSubmit,
  } = useARLFingerprintDialogState({
    fingerprint,
    onOpenChange,
    onSuccess,
    t,
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[80vh] overflow-y-auto">
        <ARLFingerprintDialogHeader t={t} isEdit={isEdit} />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <ARLFingerprintNameField
            t={t}
            register={register}
            errors={formState.errors}
          />

          <ARLFingerprintRuleField
            t={t}
            register={register}
            errors={formState.errors}
          />

          <ARLFingerprintDialogFooter
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
