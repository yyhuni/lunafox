"use client"

import { ScanConfigEditorLayout } from "@/components/scan/scan-config-editor-sections"
import { useScanConfigEditorState } from "@/components/scan/scan-config-editor-state"

import type { ScanEngine } from "@/types/engine.types"

interface ScanConfigEditorProps {
  configuration: string
  onChange: (value: string) => void
  onValidationChange?: (isValid: boolean) => void
  selectedEngines?: ScanEngine[]
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
  selectedEngines = [],
  selectedCapabilities,
  isConfigEdited = false,
  disabled = false,
  showCapabilities = true,
  className,
}: ScanConfigEditorProps) {
  const state = useScanConfigEditorState({
    selectedEngines,
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
