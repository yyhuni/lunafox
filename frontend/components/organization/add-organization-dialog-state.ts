import React from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import * as z from "zod"
import { TargetValidator } from "@/lib/target-validator"
import { useCreateOrganization } from "@/hooks/use-organizations"
import { batchCreateTargets } from "@/services/target.service"
import type { Organization } from "@/types/organization.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

type UseAddOrganizationDialogStateProps = {
  onAdd?: (organization: Organization) => void
  open?: boolean
  onOpenChange?: (open: boolean) => void
  tValidation: TranslationFn
}

export function useAddOrganizationDialogState({
  onAdd,
  open: externalOpen,
  onOpenChange: externalOnOpenChange,
  tValidation,
}: UseAddOrganizationDialogStateProps) {
  const formSchema = React.useMemo(() => z.object({
    name: z.string()
      .min(2, { message: tValidation("nameMin", { min: 2 }) })
      .max(50, { message: tValidation("nameMax", { max: 50 }) }),
    description: z.string().max(200, { message: tValidation("descMax", { max: 200 }) }).optional(),
    targets: z.string().optional(),
  }), [tValidation])

  type FormValues = z.infer<typeof formSchema>

  const [internalOpen, setInternalOpen] = React.useState(false)
  const open = externalOpen !== undefined ? externalOpen : internalOpen
  const setOpen = externalOnOpenChange || setInternalOpen

  const lineNumbersRef = React.useRef<HTMLDivElement | null>(null)
  const textareaRef = React.useRef<HTMLTextAreaElement | null>(null)

  const createOrganization = useCreateOrganization()
  const [isCreatingTargets, setIsCreatingTargets] = React.useState(false)

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      description: "",
      targets: "",
    },
  })

  const targetsText = form.watch("targets") || ""

  const targetValidation = React.useMemo(() => {
    const lines = targetsText
      .split("\n")
      .map((item) => item.trim())
      .filter((item) => item.length > 0)

    if (lines.length === 0) {
      return { count: 0, invalid: [] as Array<{ index: number; originalTarget: string; error: string; type: string }> }
    }

    const results = TargetValidator.validateTargetBatch(lines)
    const invalid = results
      .filter((result) => !result.isValid)
      .map((result) => ({
        index: result.index,
        originalTarget: result.originalTarget,
        error: result.error || tValidation("targetInvalid"),
        type: result.type,
      }))

    return { count: lines.length, invalid }
  }, [targetsText, tValidation])

  const handleTextareaScroll = (event: React.UIEvent<HTMLTextAreaElement>) => {
    if (lineNumbersRef.current) {
      lineNumbersRef.current.scrollTop = event.currentTarget.scrollTop
    }
  }

  const onSubmit = React.useCallback((values: FormValues) => {
    if (targetValidation.invalid.length > 0) return

    createOrganization.mutate(
      {
        name: values.name.trim(),
        description: values.description?.trim() || "",
      },
      {
        onSuccess: async (newOrganization) => {
          if (values.targets && values.targets.trim()) {
            const targetList = values.targets
              .split("\n")
              .map((line) => line.trim())
              .filter((line) => line.length > 0)
              .map((name) => ({ name }))

            if (targetList.length > 0) {
              setIsCreatingTargets(true)
              try {
                await batchCreateTargets({ targets: targetList, organizationId: newOrganization.id })
              } finally {
                setIsCreatingTargets(false)
              }
            }
          }
          form.reset()
          setOpen(false)
          onAdd?.(newOrganization)
        },
      }
    )
  }, [createOrganization, form, onAdd, setOpen, targetValidation.invalid.length])

  const handleOpenChange = React.useCallback((nextOpen: boolean) => {
    if (!createOrganization.isPending && !isCreatingTargets) {
      setOpen(nextOpen)
      if (!nextOpen) form.reset()
    }
  }, [createOrganization.isPending, form, isCreatingTargets, setOpen])

  const isFormValid = form.formState.isValid && targetValidation.invalid.length === 0
  const isSubmitting = createOrganization.isPending || isCreatingTargets

  return {
    form,
    open,
    handleOpenChange,
    lineNumbersRef,
    textareaRef,
    targetValidation,
    isFormValid,
    isSubmitting,
    createOrganization,
    handleTextareaScroll,
    onSubmit,
  }
}
