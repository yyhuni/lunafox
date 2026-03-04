"use client"

import { ScanConfigEditorLayout } from "@/components/scan/scan-config-editor-sections"
import { useScanConfigEditorState } from "@/components/scan/scan-config-editor-state"

import type { ScanWorkflow } from "@/types/workflow.types"

interface ScanConfigEditorProps {
  configuration: string
  onChange: (value: string) => void
  onValidationChange?: (isValid: boolean) => void
  selectedWorkflows?: ScanWorkflow[]
  selectedCapabilities?: string[]
  isConfigEdited?: boolean
  disabled?: boolean
  showCapabilities?: boolean
  className?: string
}

export function ScanConfigEditor({
  configuration,
  onChange,
  onValidationChange,
  selectedWorkflows = [],
  selectedCapabilities,
  isConfigEdited = false,
  disabled = false,
  showCapabilities = true,
  className,
}: ScanConfigEditorProps) {
  const state = useScanConfigEditorState({
    selectedWorkflows,
    selectedCapabilities,
  })

  return (
    <ScanConfigEditorLayout
      state={state}
      configuration={configuration}
      onChange={onChange}
      onValidationChange={onValidationChange}
      isConfigEdited={isConfigEdited}
      disabled={disabled}
      showCapabilities={showCapabilities}
      className={className}
    />
  )
}
