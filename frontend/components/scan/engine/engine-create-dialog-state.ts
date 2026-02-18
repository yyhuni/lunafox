import React from "react"
import * as yaml from "js-yaml"
import { toast } from "sonner"
import { usePresetEngines } from "@/hooks/use-engines"
import type { PresetEngine } from "@/types/engine.types"

export type EngineYamlError = {
  message: string
  line?: number
  column?: number
} | null

type UseEngineCreateDialogStateProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSave?: (name: string, yamlContent: string) => Promise<void>
  preSelectedPreset?: PresetEngine
  t: (key: string, params?: Record<string, string | number | Date>) => string
  tToast: (key: string, params?: Record<string, string | number | Date>) => string
}

export function useEngineCreateDialogState({
  open,
  onOpenChange,
  onSave,
  preSelectedPreset,
  t,
  tToast,
}: UseEngineCreateDialogStateProps) {
  const [step, setStep] = React.useState<1 | 2>(1)
  const [selectedPreset, setSelectedPreset] = React.useState<PresetEngine | null>(null)
  const [engineName, setEngineName] = React.useState("")
  const [yamlContent, setYamlContent] = React.useState("")
  const [isSubmitting, setIsSubmitting] = React.useState(false)
  const [yamlError, setYamlError] = React.useState<EngineYamlError>(null)

  const { data: presetEngines = [] } = usePresetEngines()

  React.useEffect(() => {
    if (open) {
      if (preSelectedPreset) {
        setSelectedPreset(preSelectedPreset)
        setEngineName(`${preSelectedPreset.name} (Copy)`)
        setYamlContent(preSelectedPreset.configuration)
        setStep(2)
      } else {
        setStep(1)
        setSelectedPreset(null)
        setEngineName("")
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
    if (!engineName.trim()) {
      toast.error(tToast("engineNameRequired"))
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
        await onSave(engineName, yamlContent)
      } else {
        await new Promise((resolve) => setTimeout(resolve, 1000))
      }

      toast.success(tToast("engineCreateSuccess"), {
        description: tToast("engineCreateSuccessDesc", { name: engineName }),
      })
      onOpenChange(false)
    } catch (error) {
      toast.error(tToast("engineCreateFailed"), {
        description: error instanceof Error ? error.message : tToast("unknownError"),
      })
    } finally {
      setIsSubmitting(false)
    }
  }, [engineName, onOpenChange, onSave, tToast, validateYaml, yamlContent, yamlError?.message])

  const handleNextStep = React.useCallback(() => {
    if (!selectedPreset) return
    setEngineName(`${selectedPreset.name} (Copy)`)
    setYamlContent(selectedPreset.configuration)
    setStep(2)
  }, [selectedPreset])

  const handleBackToStep1 = React.useCallback(() => {
    if (!preSelectedPreset) {
      setStep(1)
    }
  }, [preSelectedPreset])

  const handleClose = React.useCallback(() => {
    if (step === 2 && (engineName.trim() || yamlContent)) {
      const confirmed = window.confirm(t("confirmClose"))
      if (!confirmed) return
    }
    onOpenChange(false)
  }, [engineName, onOpenChange, step, t, yamlContent])

  return {
    step,
    presetEngines,
    selectedPreset,
    setSelectedPreset,
    engineName,
    setEngineName,
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
