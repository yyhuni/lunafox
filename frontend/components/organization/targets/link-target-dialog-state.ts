import React, { useMemo, useRef, useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import * as z from "zod"
import { TargetValidator } from "@/lib/target-validator"
import { useBatchCreateTargets } from "@/hooks/use-targets"
import type { BatchCreateResponse } from "@/types/api-response.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

type UseLinkTargetDialogStateProps = {
  organizationId: number
  onAdd?: (result: BatchCreateResponse) => void
  open?: boolean
  onOpenChange?: (open: boolean) => void
  t: TranslationFn
}

export function useLinkTargetDialogState({
  organizationId,
  onAdd,
  open: externalOpen,
  onOpenChange: externalOnOpenChange,
  t,
}: UseLinkTargetDialogStateProps) {
  const formSchema = useMemo(() => z.object({
    targets: z.string()
      .min(1, { message: t("validation.required") })
      .refine(
        (value) => {
          const lines = value.split("\n").map((item) => item.trim()).filter((item) => item.length > 0)
          return lines.length > 0
        },
        { message: t("validation.required") }
      ),
  }), [t])

  const [internalOpen, setInternalOpen] = useState(false)
  const open = externalOpen !== undefined ? externalOpen : internalOpen
  const setOpen = externalOnOpenChange || setInternalOpen

  const lineNumbersRef = useRef<HTMLDivElement | null>(null)
  const textareaRef = useRef<HTMLTextAreaElement | null>(null)

  const batchCreateTargets = useBatchCreateTargets()

  type FormValues = z.infer<typeof formSchema>

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      targets: "",
    },
  })

  const targetsText = form.watch("targets")

  const targetValidation = useMemo(() => {
    const lines = targetsText
      .split("\n")
      .map((item) => item.trim())
      .filter((item) => item.length > 0)

    if (lines.length === 0) {
      return {
        count: 0,
        invalid: [],
      }
    }

    const results = TargetValidator.validateTargetBatch(lines)
    const invalid = results
      .filter((result) => !result.isValid)
      .map((result) => ({
        index: result.index,
        originalTarget: result.originalTarget,
        error: result.error || t("validation.invalidFormat"),
        type: result.type,
      }))

    return {
      count: lines.length,
      invalid,
    }
  }, [targetsText, t])

  const onSubmit = (values: FormValues) => {
    if (targetValidation.invalid.length > 0) {
      return
    }

    const targetList = values.targets
      .split("\n")
      .map((line) => line.trim())
      .filter((line) => line.length > 0)
      .map((name) => ({
        name,
      }))

    if (targetList.length === 0) {
      return
    }

    batchCreateTargets.mutate(
      {
        targets: targetList,
        organizationId,
      },
      {
        onSuccess: (batchCreateResult) => {
          form.reset()
          setOpen(false)

          if (onAdd) {
            const adaptedResult: BatchCreateResponse = {
              message: batchCreateResult.message,
              requestedCount:
                batchCreateResult.createdCount +
                batchCreateResult.reusedCount +
                batchCreateResult.failedCount,
              createdCount: batchCreateResult.createdCount,
              existedCount: batchCreateResult.reusedCount,
              skippedCount: 0,
              skippedDomains: batchCreateResult.failedTargets.map((item) => ({
                name: item.name,
                reason: item.reason,
              })),
            }

            onAdd(adaptedResult)
          }
        },
      }
    )
  }

  const handleOpenChange = (nextOpen: boolean) => {
    if (!batchCreateTargets.isPending) {
      setOpen(nextOpen)
      if (!nextOpen) {
        form.reset()
      }
    }
  }

  const isFormValid = form.formState.isValid && targetValidation.invalid.length === 0

  const handleTextareaScroll = (event: React.UIEvent<HTMLTextAreaElement>) => {
    if (lineNumbersRef.current) {
      lineNumbersRef.current.scrollTop = event.currentTarget.scrollTop
    }
  }

  return {
    form,
    open,
    handleOpenChange,
    lineNumbersRef,
    textareaRef,
    targetValidation,
    isFormValid,
    handleTextareaScroll,
    batchCreateTargets,
    onSubmit,
  }
}
