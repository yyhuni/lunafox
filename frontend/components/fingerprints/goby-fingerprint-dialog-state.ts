import React from "react"
import { useForm, useFieldArray } from "react-hook-form"
import { toast } from "sonner"
import {
  useCreateGobyFingerprint,
  useUpdateGobyFingerprint,
} from "@/hooks/use-fingerprints"
import type { GobyFingerprint, GobyRule } from "@/types/fingerprint.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

export type GobyFingerprintFormData = {
  name: string
  logic: string
  rule: GobyRule[]
}

type UseGobyFingerprintDialogStateProps = {
  fingerprint?: GobyFingerprint | null
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
  t: TranslationFn
}

const DEFAULT_RULE: GobyRule = { label: "title", feature: "", is_equal: true }

const buildFormValues = (fingerprint?: GobyFingerprint | null): GobyFingerprintFormData => ({
  name: fingerprint?.name || "",
  logic: fingerprint?.logic || "a",
  rule: fingerprint?.rule?.length ? fingerprint.rule : [DEFAULT_RULE],
})

export function useGobyFingerprintDialogState({
  fingerprint,
  onOpenChange,
  onSuccess,
  t,
}: UseGobyFingerprintDialogStateProps) {
  const isEdit = !!fingerprint

  const createMutation = useCreateGobyFingerprint()
  const updateMutation = useUpdateGobyFingerprint()

  const form = useForm<GobyFingerprintFormData>({
    defaultValues: buildFormValues(),
  })

  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: "rule",
  })

  React.useEffect(() => {
    form.reset(buildFormValues(fingerprint))
  }, [fingerprint, form])

  const onSubmit = React.useCallback(async (data: GobyFingerprintFormData) => {
    if (data.rule.length === 0) {
      toast.error(t("form.logicRequired"))
      return
    }

    const payload = {
      name: data.name.trim(),
      logic: data.logic.trim(),
      rule: data.rule,
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
      const message = error instanceof Error ? error.message : ""
      toast.error(message || (isEdit ? t("toast.updateFailed") : t("toast.createFailed")))
    }
  }, [createMutation, fingerprint, isEdit, onOpenChange, onSuccess, t, updateMutation])

  return {
    isEdit,
    register: form.register,
    handleSubmit: form.handleSubmit,
    setValue: form.setValue,
    watch: form.watch,
    formState: form.formState,
    fields,
    append,
    remove,
    onSubmit,
  }
}
