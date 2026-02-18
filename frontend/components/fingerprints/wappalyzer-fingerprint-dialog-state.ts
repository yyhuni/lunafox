import React from "react"
import { useForm } from "react-hook-form"
import { toast } from "sonner"
import {
  useCreateWappalyzerFingerprint,
  useUpdateWappalyzerFingerprint,
} from "@/hooks/use-fingerprints"
import { getErrorMessage } from "@/lib/error-utils"
import type { WappalyzerFingerprint } from "@/types/fingerprint.types"

export type WappalyzerFingerprintFormData = {
  name: string
  cats: string
  description: string
  website: string
  cpe: string
  cookies: string
  headers: string
  scriptSrc: string
  js: string
  meta: string
  html: string
  implies: string
}

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

type UseWappalyzerFingerprintDialogStateProps = {
  fingerprint?: WappalyzerFingerprint | null
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
  t: TranslationFn
}

const buildFormValues = (fingerprint?: WappalyzerFingerprint | null): WappalyzerFingerprintFormData => ({
  name: fingerprint?.name || "",
  cats: fingerprint?.cats?.join(", ") || "",
  description: fingerprint?.description || "",
  website: fingerprint?.website || "",
  cpe: fingerprint?.cpe || "",
  cookies: JSON.stringify(fingerprint?.cookies || {}, null, 2),
  headers: JSON.stringify(fingerprint?.headers || {}, null, 2),
  scriptSrc: fingerprint?.scriptSrc?.join(", ") || "",
  js: fingerprint?.js?.join(", ") || "",
  meta: JSON.stringify(fingerprint?.meta || {}, null, 2),
  html: fingerprint?.html?.join(", ") || "",
  implies: fingerprint?.implies?.join(", ") || "",
})

const parseArray = (value: string): string[] => value
  .split(",")
  .map((item) => item.trim())
  .filter((item) => item.length > 0)

const parseNumberArray = (value: string): number[] => value
  .split(",")
  .map((item) => Number.parseInt(item.trim(), 10))
  .filter((item) => !Number.isNaN(item))

const parseJson = (value: string): Record<string, unknown> => {
  try {
    return JSON.parse(value)
  } catch {
    return {}
  }
}

const parseStringRecord = (value: string): Record<string, string> => {
  const raw = parseJson(value)
  return Object.fromEntries(
    Object.entries(raw).filter(([, entry]) => typeof entry === "string")
  ) as Record<string, string>
}

const parseStringArrayRecord = (value: string): Record<string, string[]> => {
  const raw = parseJson(value)
  const output: Record<string, string[]> = {}

  for (const [key, entry] of Object.entries(raw)) {
    if (Array.isArray(entry)) {
      const items = entry.filter((item): item is string => typeof item === "string")
      if (items.length > 0) {
        output[key] = items
      }
    }
  }

  return output
}

export function useWappalyzerFingerprintDialogState({
  fingerprint,
  onOpenChange,
  onSuccess,
  t,
}: UseWappalyzerFingerprintDialogStateProps) {
  const isEdit = !!fingerprint

  const createMutation = useCreateWappalyzerFingerprint()
  const updateMutation = useUpdateWappalyzerFingerprint()

  const form = useForm<WappalyzerFingerprintFormData>({
    defaultValues: buildFormValues(),
  })

  React.useEffect(() => {
    form.reset(buildFormValues(fingerprint))
  }, [fingerprint, form])

  const onSubmit = React.useCallback(async (data: WappalyzerFingerprintFormData) => {
    const payload = {
      name: data.name.trim(),
      cats: parseNumberArray(data.cats),
      description: data.description.trim(),
      website: data.website.trim(),
      cpe: data.cpe.trim(),
      cookies: parseStringRecord(data.cookies),
      headers: parseStringRecord(data.headers),
      scriptSrc: parseArray(data.scriptSrc),
      js: parseArray(data.js),
      meta: parseStringArrayRecord(data.meta),
      html: parseArray(data.html),
      implies: parseArray(data.implies),
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
