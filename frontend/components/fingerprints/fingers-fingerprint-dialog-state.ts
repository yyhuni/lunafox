import React from "react"
import { useForm } from "react-hook-form"
import { toast } from "sonner"
import {
  useCreateFingersFingerprint,
  useUpdateFingersFingerprint,
} from "@/hooks/use-fingerprints"
import { getErrorMessage } from "@/lib/error-utils"
import type { FingersFingerprint, FingersRule } from "@/types/fingerprint.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

export type FingersFingerprintFormData = {
  name: string
  link: string
  rule: string
  tag: string
  focus: boolean
  defaultPort: string
}

type UseFingersFingerprintDialogStateProps = {
  fingerprint?: FingersFingerprint | null
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
  t: TranslationFn
}

const buildFormValues = (fingerprint?: FingersFingerprint | null): FingersFingerprintFormData => ({
  name: fingerprint?.name || "",
  link: fingerprint?.link || "",
  rule: JSON.stringify(fingerprint?.rule || [], null, 2),
  tag: (fingerprint?.tag || []).join(", "),
  focus: fingerprint?.focus || false,
  defaultPort: (fingerprint?.defaultPort || []).join(", "),
})

export function useFingersFingerprintDialogState({
  fingerprint,
  onOpenChange,
  onSuccess,
  t,
}: UseFingersFingerprintDialogStateProps) {
  const isEdit = !!fingerprint

  const createMutation = useCreateFingersFingerprint()
  const updateMutation = useUpdateFingersFingerprint()

  const form = useForm<FingersFingerprintFormData>({
    defaultValues: buildFormValues(),
  })

  React.useEffect(() => {
    form.reset(buildFormValues(fingerprint))
  }, [fingerprint, form])

  const onSubmit = React.useCallback(async (data: FingersFingerprintFormData) => {
    let ruleArray: FingersRule[]
    try {
      ruleArray = JSON.parse(data.rule)
      if (!Array.isArray(ruleArray)) {
        toast.error(t("form.ruleArrayRequired"))
        return
      }
    } catch {
      toast.error(t("form.ruleJsonInvalid"))
      return
    }

    const tagArray = data.tag
      .split(",")
      .map((item) => item.trim())
      .filter((item) => item.length > 0)

    const portArray = data.defaultPort
      .split(",")
      .map((item) => Number.parseInt(item.trim(), 10))
      .filter((item) => !Number.isNaN(item))

    const payload = {
      name: data.name.trim(),
      link: data.link.trim(),
      rule: ruleArray,
      tag: tagArray,
      focus: data.focus,
      defaultPort: portArray,
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
