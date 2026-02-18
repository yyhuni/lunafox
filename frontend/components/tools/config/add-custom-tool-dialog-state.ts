import React from "react"
import { useCreateTool, useUpdateTool } from "@/hooks/use-tools"
import { CategoryNameMap, type Tool } from "@/types/tool.types"

type FormData = {
  name: string
  description: string
  directory: string
  categoryNames: string[]
}

type UseAddCustomToolDialogStateProps = {
  tool?: Tool
  onAdd?: (tool: Tool) => void
  open?: boolean
  onOpenChange?: (open: boolean) => void
}

export function useAddCustomToolDialogState({
  tool,
  onAdd,
  open: externalOpen,
  onOpenChange: externalOnOpenChange,
}: UseAddCustomToolDialogStateProps) {
  const isEditMode = !!tool

  const [internalOpen, setInternalOpen] = React.useState(false)
  const open = externalOpen !== undefined ? externalOpen : internalOpen
  const setOpen = externalOnOpenChange || setInternalOpen

  const [formData, setFormData] = React.useState<FormData>({
    name: tool?.name || "",
    description: tool?.description || "",
    directory: tool?.directory || "",
    categoryNames: tool?.categoryNames || [],
  })

  const availableCategories = React.useMemo(() => Object.keys(CategoryNameMap), [])

  const createTool = useCreateTool()
  const updateTool = useUpdateTool()

  React.useEffect(() => {
    if (tool) {
      setFormData({
        name: tool.name || "",
        description: tool.description || "",
        directory: tool.directory || "",
        categoryNames: tool.categoryNames || [],
      })
    }
  }, [tool])

  const resetForm = React.useCallback(() => {
    setFormData({
      name: "",
      description: "",
      directory: "",
      categoryNames: [],
    })
  }, [])

  const handleOpenChange = React.useCallback((nextOpen: boolean) => {
    if (!createTool.isPending && !updateTool.isPending) {
      setOpen(nextOpen)
      if (!nextOpen) {
        resetForm()
      }
    }
  }, [createTool.isPending, resetForm, setOpen, updateTool.isPending])

  const handleCategoryToggle = React.useCallback((categoryName: string) => {
    setFormData((prev) => {
      const isSelected = prev.categoryNames.includes(categoryName)
      return {
        ...prev,
        categoryNames: isSelected
          ? prev.categoryNames.filter((item) => item !== categoryName)
          : [...prev.categoryNames, categoryName],
      }
    })
  }, [])

  const handleCategoryRemove = React.useCallback((categoryName: string) => {
    setFormData((prev) => ({
      ...prev,
      categoryNames: prev.categoryNames.filter((item) => item !== categoryName),
    }))
  }, [])

  const setFormField = React.useCallback((field: keyof FormData, value: string) => {
    setFormData((prev) => ({
      ...prev,
      [field]: value,
    }))
  }, [])

  const handleSubmit = React.useCallback((event: React.FormEvent) => {
    event.preventDefault()

    if (!formData.name.trim() || !formData.directory.trim()) {
      return
    }

    const toolData = {
      name: formData.name.trim(),
      type: "custom" as const,
      description: formData.description.trim() || undefined,
      directory: formData.directory.trim(),
      categoryNames: formData.categoryNames.length > 0 ? formData.categoryNames : undefined,
    }

    const onSuccess = (response: { tool?: Tool }) => {
      resetForm()
      setOpen(false)
      if (onAdd && response?.tool) {
        onAdd(response.tool)
      }
    }

    if (isEditMode && tool?.id) {
      updateTool.mutate({ id: tool.id, data: toolData }, { onSuccess })
    } else {
      createTool.mutate(toolData, { onSuccess })
    }
  }, [createTool, formData, isEditMode, onAdd, resetForm, setOpen, tool?.id, updateTool])

  const isFormValid = formData.name.trim().length > 0 && formData.directory.trim().length > 0
  const isSubmitting = createTool.isPending || updateTool.isPending

  return {
    open,
    formData,
    isEditMode,
    availableCategories,
    createTool,
    updateTool,
    isFormValid,
    isSubmitting,
    handleOpenChange,
    handleCategoryToggle,
    handleCategoryRemove,
    handleSubmit,
    setFormField,
  }
}
