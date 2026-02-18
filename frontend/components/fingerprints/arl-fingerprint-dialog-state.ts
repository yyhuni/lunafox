import React from "react"
import { useForm } from "react-hook-form"
import { toast } from "sonner"
import {
  useCreateARLFingerprint,
  useUpdateARLFingerprint,
} from "@/hooks/use-fingerprints"
import { getErrorMessage } from "@/lib/error-utils"
import type { ARLFingerprint } from "@/types/fingerprint.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

export type ARLFingerprintFormData = {
  name: string
  rule: string
}

type UseARLFingerprintDialogStateProps = {
  fingerprint?: ARLFingerprint | null
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
  t: TranslationFn
}

const buildFormValues = (fingerprint?: ARLFingerprint | null): ARLFingerprintFormData => ({
  name: fingerprint?.name || "",
  rule: fingerprint?.rule || "",
})

export function useARLFingerprintDialogState({
  fingerprint,
  onOpenChange,
  onSuccess,
  t,
}: UseARLFingerprintDialogStateProps) {
  const isEdit = !!fingerprint

  const createMutation = useCreateARLFingerprint()
  const updateMutation = useUpdateARLFingerprint()

  const form = useForm<ARLFingerprintFormData>({
    defaultValues: buildFormValues(),
  })

  React.useEffect(() => {
    form.reset(buildFormValues(fingerprint))
  }, [fingerprint, form])

  const onSubmit = React.useCallback(async (data: ARLFingerprintFormData) => {
    const payload = {
      name: data.name.trim(),
      rule: data.rule.trim(),
    }

    try {
      if (isEdit && fingerprint) {
        await updateMutation.mutateAsync({ id: fingerprint.id, data: payload })
        toast.success(t("toast.updateSuccess"))
      } else {
        await createMutation.mutateAsync(payload)
        toast.success(t("toast.createSuccess"))
      }
      onOpenChange(false)
      onSuccess?.()
    } catch (error) {
      toast.error(getErrorMessage(error) || (isEdit ? t("toast.updateFailed") : t("toast.createFailed")))
    }
  }, [createMutation, fingerprint, isEdit, onOpenChange, onSuccess, t, updateMutation])

  return {
    isEdit,
    register: form.register,
    handleSubmit: form.handleSubmit,
    formState: form.formState,
    onSubmit,
  }
}
