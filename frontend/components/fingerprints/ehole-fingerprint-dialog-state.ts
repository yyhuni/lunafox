import React from "react"
import { useForm } from "react-hook-form"
import { toast } from "sonner"
import {
  useCreateEholeFingerprint,
  useUpdateEholeFingerprint,
} from "@/hooks/use-fingerprints"
import { getErrorMessage } from "@/lib/error-utils"
import type { EholeFingerprint } from "@/types/fingerprint.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

export type EholeFingerprintFormData = {
  cms: string
  method: string
  location: string
  keyword: string
  type: string
  isImportant: boolean
}

type UseEholeFingerprintDialogStateProps = {
  fingerprint?: EholeFingerprint | null
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
  t: TranslationFn
}

const buildFormValues = (fingerprint?: EholeFingerprint | null): EholeFingerprintFormData => ({
  cms: fingerprint?.cms || "",
  method: fingerprint?.method || "keyword",
  location: fingerprint?.location || "body",
  keyword: fingerprint?.keyword?.join(", ") || "",
  type: fingerprint?.type || "-",
  isImportant: fingerprint?.isImportant ?? false,
})

export function useEholeFingerprintDialogState({
  fingerprint,
  onOpenChange,
  onSuccess,
  t,
}: UseEholeFingerprintDialogStateProps) {
  const isEdit = !!fingerprint

  const createMutation = useCreateEholeFingerprint()
  const updateMutation = useUpdateEholeFingerprint()

  const form = useForm<EholeFingerprintFormData>({
    defaultValues: buildFormValues(),
  })

  React.useEffect(() => {
    form.reset(buildFormValues(fingerprint))
  }, [fingerprint, form])

  const onSubmit = React.useCallback(async (data: EholeFingerprintFormData) => {
    const keywordArray = data.keyword
      .split(",")
      .map((item) => item.trim())
      .filter((item) => item.length > 0)

    if (keywordArray.length === 0) {
      toast.error(t("form.keywordRequired"))
      return
    }

    const payload = {
      cms: data.cms.trim(),
      method: data.method,
      location: data.location,
      keyword: keywordArray,
      type: data.type || "-",
      isImportant: data.isImportant,
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
    setValue: form.setValue,
    watch: form.watch,
    formState: form.formState,
    onSubmit,
  }
}
