"use client"

import React from "react"
import { useTranslations } from "next-intl"
import { Dialog, DialogContent } from "@/components/ui/dialog"
import { useFingersFingerprintDialogState } from "@/components/fingerprints/fingers-fingerprint-dialog-state"
import {
  FingersFingerprintDialogHeader,
  FingersFingerprintNameField,
  FingersFingerprintLinkField,
  FingersFingerprintRuleField,
  FingersFingerprintTagsFocus,
  FingersFingerprintDefaultPort,
  FingersFingerprintDialogFooter,
} from "@/components/fingerprints/fingers-fingerprint-dialog-sections"
import type { FingersFingerprint } from "@/types/fingerprint.types"

interface FingersFingerprintDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  fingerprint?: FingersFingerprint | null
  onSuccess?: () => void
}

export function FingersFingerprintDialog({
  open,
  onOpenChange,
  fingerprint,
  onSuccess,
}: FingersFingerprintDialogProps) {
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
  } = useFingersFingerprintDialogState({
    fingerprint,
    onOpenChange,
    onSuccess,
    t,
  })

  const focus = watch("focus")

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[80vh] overflow-y-auto">
        <FingersFingerprintDialogHeader t={t} isEdit={isEdit} />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <FingersFingerprintNameField
            t={t}
            register={register}
            errors={formState.errors}
          />

          <FingersFingerprintLinkField
            t={t}
            register={register}
          />

          <FingersFingerprintRuleField
            t={t}
            register={register}
            errors={formState.errors}
          />

          <FingersFingerprintTagsFocus
            t={t}
            register={register}
            focus={focus}
            onFocusChange={(value) => setValue("focus", value)}
          />

          <FingersFingerprintDefaultPort
            t={t}
            register={register}
          />

          <FingersFingerprintDialogFooter
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
