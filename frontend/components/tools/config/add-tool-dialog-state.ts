import React from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import * as z from "zod"
import { useCreateTool, useUpdateTool } from "@/hooks/use-tools"
import type { Tool } from "@/types/tool.types"
import { CategoryNameMap } from "@/types/tool.types"

interface UseAddToolDialogStateProps {
  tool?: Tool
  onAdd?: (tool: Tool) => void
  externalOpen?: boolean
  externalOnOpenChange?: (open: boolean) => void
  t: (key: string, params?: Record<string, string | number | Date>) => string
}

/**
 * Auto-generate version query command based on tool name and install command
 */
function generateVersionCommand(toolName: string, installCommand: string): string {
  if (!toolName) return ""

  const lowerName = toolName.toLowerCase().trim()
  const lowerInstall = installCommand.toLowerCase()

  if (lowerInstall.includes("python") || lowerInstall.includes(".py")) {
    return `python ${lowerName}.py -v`
  }

  if (lowerInstall.includes("go install") || lowerInstall.includes("go get")) {
    return `${lowerName} -version`
  }

  return `${lowerName} --version`
}

export type AddToolFormValues = {
  name: string
  repoUrl?: string
  version?: string
  description?: string
  categoryNames: string[]
  installCommand: string
  updateCommand: string
  versionCommand: string
}

export function useAddToolDialogState({
  tool,
  onAdd,
  externalOpen,
  externalOnOpenChange,
  t,
}: UseAddToolDialogStateProps) {
  const [internalOpen, setInternalOpen] = React.useState(false)
  const open = externalOpen !== undefined ? externalOpen : internalOpen
  const setOpen = externalOnOpenChange || setInternalOpen

  const isEditMode = Boolean(tool)
  const availableCategories = React.useMemo(() => Object.keys(CategoryNameMap), [])

  const createTool = useCreateTool()
  const updateTool = useUpdateTool()

  const formSchema = React.useMemo(() => z.object({
    name: z.string()
      .min(2, { message: t("toolNameMin") })
      .max(255, { message: t("toolNameMax") }),
    repoUrl: z.string().optional().or(z.literal("")),
    version: z.string().max(100).optional().or(z.literal("")),
    description: z.string().max(1000).optional().or(z.literal("")),
    categoryNames: z.array(z.string()),
    installCommand: z.string().min(1, { message: t("installCommandRequired") }),
    updateCommand: z.string().min(1, { message: t("updateCommandRequired") }),
    versionCommand: z.string().min(1, { message: t("versionCommandRequired") }),
  }), [t])

  const form = useForm<AddToolFormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: tool?.name || "",
      repoUrl: tool?.repoUrl || "",
      version: tool?.version || "",
      description: tool?.description || "",
      categoryNames: tool?.categoryNames || [],
      installCommand: tool?.installCommand || "",
      updateCommand: tool?.updateCommand || "",
      versionCommand: tool?.versionCommand || "",
    },
  })

  React.useEffect(() => {
    if (tool) {
      form.reset({
        name: tool.name || "",
        repoUrl: tool.repoUrl || "",
        version: tool.version || "",
        description: tool.description || "",
        categoryNames: tool.categoryNames || [],
        installCommand: tool.installCommand || "",
        updateCommand: tool.updateCommand || "",
        versionCommand: tool.versionCommand || "",
      })
    }
  }, [tool, form])

  const watchName = form.watch("name")
  const watchInstallCommand = form.watch("installCommand")
  const watchVersionCommand = form.watch("versionCommand")
  const watchCategoryNames = form.watch("categoryNames")

  React.useEffect(() => {
    if (watchName && watchInstallCommand && !watchVersionCommand) {
      const generatedCmd = generateVersionCommand(watchName, watchInstallCommand)
      form.setValue("versionCommand", generatedCmd)
    }
  }, [watchName, watchInstallCommand, watchVersionCommand, form])

  const onSubmit = React.useCallback((values: AddToolFormValues) => {
    const toolData = {
      name: values.name.trim(),
      type: "opensource" as const,
      repoUrl: values.repoUrl?.trim() || undefined,
      version: values.version?.trim() || undefined,
      description: values.description?.trim() || undefined,
      categoryNames: values.categoryNames.length > 0 ? values.categoryNames : undefined,
      installCommand: values.installCommand.trim(),
      updateCommand: values.updateCommand.trim(),
      versionCommand: values.versionCommand.trim(),
    }

    const onSuccessCallback = (response: { tool?: Tool }) => {
      form.reset()
      setOpen(false)
      if (onAdd && response?.tool) {
        onAdd(response.tool)
      }
    }

    if (isEditMode && tool?.id) {
      updateTool.mutate(
        { id: tool.id, data: toolData },
        { onSuccess: onSuccessCallback }
      )
    } else {
      createTool.mutate(toolData, { onSuccess: onSuccessCallback })
    }
  }, [createTool, form, isEditMode, onAdd, setOpen, tool, updateTool])

  const handleCategoryToggle = React.useCallback((categoryName: string) => {
    const current = form.getValues("categoryNames")
    const isSelected = current.includes(categoryName)
    form.setValue(
      "categoryNames",
      isSelected
        ? current.filter((c) => c !== categoryName)
        : [...current, categoryName]
    )
  }, [form])

  const handleCategoryRemove = React.useCallback((categoryName: string) => {
    const current = form.getValues("categoryNames")
    form.setValue("categoryNames", current.filter((c) => c !== categoryName))
  }, [form])

  const handleOpenChange = React.useCallback((newOpen: boolean) => {
    if (!createTool.isPending && !updateTool.isPending) {
      setOpen(newOpen)
      if (!newOpen) {
        form.reset()
      }
    }
  }, [createTool.isPending, form, setOpen, updateTool.isPending])

  return {
    open,
    isEditMode,
    availableCategories,
    createTool,
    updateTool,
    form,
    watchCategoryNames,
    handleCategoryToggle,
    handleCategoryRemove,
    handleOpenChange,
    onSubmit,
  }
}
