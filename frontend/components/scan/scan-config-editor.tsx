"use client"

import { ScanConfigEditorLayout } from "@/components/scan/scan-config-editor-sections"
import { useScanConfigEditorState } from "@/components/scan/scan-config-editor-state"

import type { ScanWorkflow } from "@/types/scan-workflow.types"

interface ScanConfigEditorProps {
  configuration: string
  onChange: (value: string) => void
  onValidationChange?: (isValid: boolean) => void
  selectedScanWorkflows?: ScanWorkflow[]
  selectedCapabilities?: string[]
  isConfigEdited?: boolean
  disabled?: boolean
  showCapabilities?: boolean
  showLabel?: boolean
  className?: string
}

export function ScanConfigEditor({
  configuration,
  onChange,
  onValidationChange,
  selectedScanWorkflows = [],
  selectedCapabilities,
  isConfigEdited = false,
  disabled = false,
  showCapabilities = true,
  showLabel = true,
  className,
}: ScanConfigEditorProps) {
  const state = useScanConfigEditorState({
    selectedScanWorkflows,
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
      showLabel={showLabel}
      className={className}
    />
  )
}
