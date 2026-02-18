import React from "react"
import { useForm } from "react-hook-form"
import { toast } from "sonner"
import {
  useCreateFingerPrintHubFingerprint,
  useUpdateFingerPrintHubFingerprint,
} from "@/hooks/use-fingerprints"
import { getErrorMessage } from "@/lib/error-utils"
import type { FingerPrintHubFingerprint, FingerPrintHubHttpMatcher } from "@/types/fingerprint.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

export type FingerPrintHubFingerprintFormData = {
  fpId: string
  name: string
  author: string
  tags: string
  severity: string
  metadata: string
  http: string
  sourceFile: string
}

type UseFingerPrintHubFingerprintDialogStateProps = {
  fingerprint?: FingerPrintHubFingerprint | null
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
  t: TranslationFn
}

const buildFormValues = (fingerprint?: FingerPrintHubFingerprint | null): FingerPrintHubFingerprintFormData => ({
  fpId: fingerprint?.fpId || "",
  name: fingerprint?.name || "",
  author: fingerprint?.author || "",
  tags: fingerprint?.tags || "",
  severity: fingerprint?.severity || "info",
  metadata: JSON.stringify(fingerprint?.metadata || {}, null, 2),
  http: JSON.stringify(fingerprint?.http || [], null, 2),
  sourceFile: fingerprint?.sourceFile || "",
})

export function useFingerPrintHubFingerprintDialogState({
  fingerprint,
  onOpenChange,
  onSuccess,
  t,
}: UseFingerPrintHubFingerprintDialogStateProps) {
  const isEdit = !!fingerprint

  const createMutation = useCreateFingerPrintHubFingerprint()
  const updateMutation = useUpdateFingerPrintHubFingerprint()

  const form = useForm<FingerPrintHubFingerprintFormData>({
    defaultValues: buildFormValues(),
  })

  React.useEffect(() => {
    form.reset(buildFormValues(fingerprint))
  }, [fingerprint, form])

  const onSubmit = React.useCallback(async (data: FingerPrintHubFingerprintFormData) => {
    let metadataObj: Record<string, unknown>
    try {
      metadataObj = JSON.parse(data.metadata)
      if (typeof metadataObj !== "object" || Array.isArray(metadataObj)) {
        toast.error(t("form.metadataObjectRequired"))
        return
      }
    } catch {
      toast.error(t("form.metadataJsonInvalid"))
      return
    }

    let httpArray: FingerPrintHubHttpMatcher[]
    try {
      httpArray = JSON.parse(data.http)
      if (!Array.isArray(httpArray)) {
        toast.error(t("form.httpArrayRequired"))
        return
      }
    } catch {
      toast.error(t("form.httpJsonInvalid"))
      return
    }

    const payload = {
      fpId: data.fpId.trim(),
      name: data.name.trim(),
      author: data.author.trim(),
      tags: data.tags.trim(),
      severity: data.severity,
      metadata: metadataObj,
      http: httpArray,
      sourceFile: data.sourceFile.trim(),
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
