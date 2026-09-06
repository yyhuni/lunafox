import React from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import * as z from "zod"
import { useUpdateOrganization } from "@/hooks/use-organizations"
import type { Organization } from "@/types/organization.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

type UseEditOrganizationDialogStateProps = {
  organization: Organization
  onOpenChange: (open: boolean) => void
  onEdit: (organization: Organization) => void
  tValidation: TranslationFn
}

export function useEditOrganizationDialogState({
  organization,
  onOpenChange,
  onEdit,
  tValidation,
}: UseEditOrganizationDialogStateProps) {
  const formSchema = React.useMemo(() => z.object({
    name: z.string()
      .min(2, { message: tValidation("nameMin", { min: 2 }) })
      .max(50, { message: tValidation("nameMax", { max: 50 }) }),
    description: z.string().max(200, { message: tValidation("descMax", { max: 200 }) }).optional(),
  }), [tValidation])

  type FormValues = z.infer<typeof formSchema>

  const updateOrganization = useUpdateOrganization()

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: organization?.name || "",
      description: organization?.description || "",
    },
  })

  React.useEffect(() => {
    if (organization) {
      form.reset({
        name: organization.name || "",
        description: organization.description || "",
      })
    }
  }, [organization, form])

  const hasChanges = form.formState.isDirty
  const isFormValid = form.formState.isValid
  const isUpdating = updateOrganization.isPending

  const onSubmit = React.useCallback((values: FormValues) => {
    updateOrganization.mutate(
      {
        id: Number(organization.id),
        data: {
          name: values.name.trim(),
          description: values.description?.trim() || "",
        },
      },
      {
        onSuccess: (updatedOrganization) => {
          onEdit(updatedOrganization)
          onOpenChange(false)
        },
      }
    )
  }, [onEdit, onOpenChange, organization.id, updateOrganization])

  const handleOpenChange = React.useCallback((nextOpen: boolean) => {
    if (!updateOrganization.isPending) {
      onOpenChange(nextOpen)
    }
  }, [onOpenChange, updateOrganization.isPending])

  const handleReset = React.useCallback(() => {
    form.reset({
      name: organization.name || "",
      description: organization.description || "",
    })
  }, [form, organization.description, organization.name])

  return {
    form,
    hasChanges,
    isFormValid,
    isUpdating,
    handleOpenChange,
    handleReset,
    onSubmit,
  }
}
