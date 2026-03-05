import React from "react"
import * as yaml from "js-yaml"
import { toast } from "sonner"
import { useWorkflowProfiles } from "@/hooks/use-workflows"
import type { WorkflowProfile } from "@/types/workflow.types"

export type WorkflowYamlError = {
  message: string
  line?: number
  column?: number
} | null

type UseWorkflowCreateDialogStateProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSave?: (name: string, yamlContent: string) => Promise<void>
  preSelectedPreset?: WorkflowProfile
  t: (key: string, params?: Record<string, string | number | Date>) => string
  tToast: (key: string, params?: Record<string, string | number | Date>) => string
}

export function useWorkflowCreateDialogState({
  open,
  onOpenChange,
  onSave,
  preSelectedPreset,
  t,
  tToast,
}: UseWorkflowCreateDialogStateProps) {
  const [step, setStep] = React.useState<1 | 2>(1)
  const [selectedPreset, setSelectedPreset] = React.useState<WorkflowProfile | null>(null)
  const [workflowName, setWorkflowName] = React.useState("")
  const [yamlContent, setYamlContent] = React.useState("")
  const [isSubmitting, setIsSubmitting] = React.useState(false)
  const [yamlError, setYamlError] = React.useState<WorkflowYamlError>(null)

  const { data: presetWorkflows = [] } = useWorkflowProfiles()

  React.useEffect(() => {
    if (open) {
      if (preSelectedPreset) {
        setSelectedPreset(preSelectedPreset)
        setWorkflowName(`${preSelectedPreset.name} (Copy)`)
        setYamlContent(preSelectedPreset.configuration)
        setStep(2)
      } else {
        setStep(1)
        setSelectedPreset(null)
        setWorkflowName("")
        setYamlContent("")
      }
      setYamlError(null)
    }
  }, [open, preSelectedPreset])

  const validateYaml = React.useCallback((content: string) => {
    if (!content.trim()) {
      setYamlError(null)
      return true
    }

    try {
      yaml.load(content)
      setYamlError(null)
      return true
    } catch (error) {
      const yamlError = error as yaml.YAMLException
      setYamlError({
        message: yamlError.message,
        line: yamlError.mark?.line ? yamlError.mark.line + 1 : undefined,
        column: yamlError.mark?.column ? yamlError.mark.column + 1 : undefined,
      })
      return false
    }
  }, [])

  const handleEditorChange = React.useCallback((value: string) => {
    setYamlContent(value)
    validateYaml(value)
  }, [validateYaml])

  const handleSave = React.useCallback(async () => {
    if (!workflowName.trim()) {
      toast.error(tToast("workflowNameRequired"))
      return
    }

    if (!yamlContent.trim()) {
      toast.error(tToast("configRequired"))
      return
    }

    if (!validateYaml(yamlContent)) {
      toast.error(tToast("yamlSyntaxError"), {
        description: yamlError?.message,
      })
      return
    }

    setIsSubmitting(true)
    try {
      if (onSave) {
        await onSave(workflowName, yamlContent)
      } else {
        await new Promise((resolve) => setTimeout(resolve, 1000))
      }

      toast.success(tToast("workflowCreateSuccess"), {
        description: tToast("workflowCreateSuccessDesc", { name: workflowName }),
      })
      onOpenChange(false)
    } catch (error) {
      toast.error(tToast("workflowCreateFailed"), {
        description: error instanceof Error ? error.message : tToast("unknownError"),
      })
    } finally {
      setIsSubmitting(false)
    }
  }, [workflowName, onOpenChange, onSave, tToast, validateYaml, yamlContent, yamlError?.message])

  const handleNextStep = React.useCallback(() => {
    if (!selectedPreset) return
    setWorkflowName(`${selectedPreset.name} (Copy)`)
    setYamlContent(selectedPreset.configuration)
    setStep(2)
  }, [selectedPreset])

  const handleBackToStep1 = React.useCallback(() => {
    if (!preSelectedPreset) {
      setStep(1)
    }
  }, [preSelectedPreset])

  const handleClose = React.useCallback(() => {
    if (step === 2 && (workflowName.trim() || yamlContent)) {
      const confirmed = window.confirm(t("confirmClose"))
      if (!confirmed) return
    }
    onOpenChange(false)
  }, [workflowName, onOpenChange, step, t, yamlContent])

  return {
    step,
    presetWorkflows,
    selectedPreset,
    setSelectedPreset,
    workflowName,
    setWorkflowName,
    yamlContent,
    yamlError,
    isSubmitting,
    handleEditorChange,
    handleSave,
    handleNextStep,
    handleBackToStep1,
    handleClose,
  }
}
