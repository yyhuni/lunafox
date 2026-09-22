import React, { useMemo, useRef, useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import * as z from "zod"
import { MAX_TARGET_BATCH_SIZE, TargetValidator } from "@/lib/target-validator"
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
        (value) => TargetValidator.parseLines(value).length > 0,
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
    const lines = TargetValidator.parseLines(targetsText)

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
        lineNumber: result.lineNumber,
        originalTarget: result.originalTarget,
        error: result.error || t("validation.invalidFormat"),
        type: result.type,
      }))

    return {
      count: lines.length,
      invalid,
    }
  }, [targetsText, t])
  const isTargetBatchOverLimit = targetValidation.count > MAX_TARGET_BATCH_SIZE

  const onSubmit = (values: FormValues) => {
    const submittedTargets = TargetValidator.parseLines(values.targets)
    if (submittedTargets.length === 0 || submittedTargets.length > MAX_TARGET_BATCH_SIZE) return
    if (TargetValidator.validateTargetBatch(submittedTargets).some((result) => !result.isValid)) return

    const targetList = submittedTargets.map(({ target: name }) => ({ name }))

    batchCreateTargets.mutate(
      {
        targets: targetList,
        organizationIds: [organizationId],
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

  const isFormValid = form.formState.isValid && targetValidation.invalid.length === 0 && !isTargetBatchOverLimit

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
    isTargetBatchOverLimit,
    isFormValid,
    handleTextareaScroll,
    batchCreateTargets,
    onSubmit,
  }
}
