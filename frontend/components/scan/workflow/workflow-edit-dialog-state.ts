import React from "react"
import * as yaml from "js-yaml"
import { toastFeedback } from "@/lib/toast-helpers"
import type { ScanWorkflow } from "@/types/scan-workflow.types"

export type WorkflowEditYamlError = {
  message: string
  line?: number
  column?: number
} | null

type UseWorkflowEditDialogStateProps = {
  workflow: ScanWorkflow | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onSave?: (workflowName: string, yamlContent: string) => Promise<void>
  t: (key: string, params?: Record<string, string | number | Date>) => string
  tToast: (key: string, params?: Record<string, string | number | Date>) => string
}

const generateSampleYaml = (workflow: ScanWorkflow) => {
  const stages = workflow.stages.map((stage) => ({
    stageId: stage.stageId,
    steps: stage.steps.map((step) => ({
      stepId: step.stepId,
      engineId: step.engineId,
      profileDefaultEnabled: step.profileDefaultEnabled,
    })),
  }))
  return `# Workflow definition: ${workflow.name}\n${JSON.stringify({ stages }, null, 2)}`
}

export function useWorkflowEditDialogState({
  workflow,
  open,
  onOpenChange,
  onSave,
  t,
  tToast,
}: UseWorkflowEditDialogStateProps) {
  const [yamlContent, setYamlContent] = React.useState("")
  const [isSubmitting, setIsSubmitting] = React.useState(false)
  const [hasChanges, setHasChanges] = React.useState(false)
  const [yamlError, setYamlError] = React.useState<WorkflowEditYamlError>(null)

  React.useEffect(() => {
    if (workflow && open) {
      const content = generateSampleYaml(workflow)
      setYamlContent(content)
      setHasChanges(false)
      setYamlError(null)
    }
  }, [workflow, open])

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
    setHasChanges(true)
    validateYaml(value)
  }, [validateYaml])

  const handleSave = React.useCallback(async () => {
    if (!workflow) return

    if (!yamlContent.trim()) {
      toastFeedback.error(tToast("configRequired"))
      return
    }

    if (!validateYaml(yamlContent)) {
      toastFeedback.error(tToast("yamlSyntaxError"), {
        description: yamlError?.message,
      })
      return
    }

    setIsSubmitting(true)
    try {
      if (onSave) {
        await onSave(workflow.name, yamlContent)
      } else {
        await new Promise((resolve) => setTimeout(resolve, 1000))
      }

      setHasChanges(false)
      onOpenChange(false)
    } catch {
      // Error toast handled upstream
    } finally {
      setIsSubmitting(false)
    }
  }, [workflow, onOpenChange, onSave, tToast, validateYaml, yamlContent, yamlError?.message])

  const handleClose = React.useCallback(() => {
    if (hasChanges) {
      const confirmed = window.confirm(t("confirmClose"))
      if (!confirmed) return
    }
    onOpenChange(false)
  }, [hasChanges, onOpenChange, t])

  return {
    yamlContent,
    isSubmitting,
    hasChanges,
    yamlError,
    handleEditorChange,
    handleSave,
    handleClose,
  }
}
