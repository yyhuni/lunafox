"use client"

import React from "react"
import { useTranslations } from "next-intl"
import { Dialog, DialogContent } from "@/components/ui/dialog"
import { useEholeFingerprintDialogState } from "@/components/fingerprints/ehole-fingerprint-dialog-state"
import {
  EholeFingerprintDialogHeader,
  EholeFingerprintCmsField,
  EholeFingerprintMethodLocation,
  EholeFingerprintKeywordField,
  EholeFingerprintTypeImportant,
  EholeFingerprintDialogFooter,
} from "@/components/fingerprints/ehole-fingerprint-dialog-sections"
import type { EholeFingerprint } from "@/types/fingerprint.types"

interface EholeFingerprintDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  fingerprint?: EholeFingerprint | null
  onSuccess?: () => void
}

export function EholeFingerprintDialog({
  open,
  onOpenChange,
  fingerprint,
  onSuccess,
}: EholeFingerprintDialogProps) {
  const t = useTranslations("tools.fingerprints")
  const tCommon = useTranslations("common.actions")
  const tColumns = useTranslations("columns.fingerprint")

  const {
    isEdit,
    register,
    handleSubmit,
    setValue,
    watch,
    formState,
    onSubmit,
  } = useEholeFingerprintDialogState({
    fingerprint,
    onOpenChange,
    onSuccess,
    t,
  })

  const method = watch("method")
  const location = watch("location")
  const isImportant = watch("isImportant")

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[80vh] overflow-y-auto">
        <EholeFingerprintDialogHeader t={t} isEdit={isEdit} />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <EholeFingerprintCmsField
            t={t}
            tColumns={tColumns}
            register={register}
            errors={formState.errors}
          />

          <EholeFingerprintMethodLocation
            t={t}
            tColumns={tColumns}
            method={method}
            location={location}
            onMethodChange={(value) => setValue("method", value)}
            onLocationChange={(value) => setValue("location", value)}
          />

          <EholeFingerprintKeywordField
            t={t}
            tColumns={tColumns}
            register={register}
            errors={formState.errors}
          />

          <EholeFingerprintTypeImportant
            t={t}
            tColumns={tColumns}
            register={register}
            isImportant={isImportant}
            onImportantChange={(value) => setValue("isImportant", value)}
          />

          <EholeFingerprintDialogFooter
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
