"use client"

import React from "react"
import { useTranslations } from "next-intl"
import { Dialog, DialogContent } from "@/components/ui/dialog"
import { useGobyFingerprintDialogState } from "@/components/fingerprints/goby-fingerprint-dialog-state"
import {
  GobyFingerprintDialogHeader,
  GobyFingerprintBasicFields,
  GobyFingerprintRulesSection,
  GobyFingerprintDialogFooter,
} from "@/components/fingerprints/goby-fingerprint-dialog-sections"
import type { GobyFingerprint } from "@/types/fingerprint.types"

interface GobyFingerprintDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  fingerprint?: GobyFingerprint | null
  onSuccess?: () => void
}

export function GobyFingerprintDialog({
  open,
  onOpenChange,
  fingerprint,
  onSuccess,
}: GobyFingerprintDialogProps) {
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
    fields,
    append,
    remove,
    onSubmit,
  } = useGobyFingerprintDialogState({
    fingerprint,
    onOpenChange,
    onSuccess,
    t,
  })

  const watchedRules = watch("rule")

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[80vh] overflow-y-auto">
        <GobyFingerprintDialogHeader t={t} isEdit={isEdit} />

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <GobyFingerprintBasicFields
            t={t}
            tColumns={tColumns}
            register={register}
            errors={formState.errors}
          />

          <GobyFingerprintRulesSection
            t={t}
            tCommon={tCommon}
            tColumns={tColumns}
            fields={fields}
            register={register}
            setValue={setValue}
            watchedRules={watchedRules}
            append={append}
            remove={remove}
          />

          <GobyFingerprintDialogFooter
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
