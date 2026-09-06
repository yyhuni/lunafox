import React from "react"

import { CAPABILITY_CONFIG } from "@/lib/scan-workflow-config"

import type { ScanWorkflow } from "@/types/scan-workflow.types"

interface ScanConfigEditorStateOptions {
  selectedScanWorkflows?: ScanWorkflow[]
  selectedCapabilities?: string[]
}

export function useScanConfigEditorState({
  selectedScanWorkflows = [],
  selectedCapabilities: propCapabilities,
}: ScanConfigEditorStateOptions) {
  const capabilities = React.useMemo(() => {
    if (propCapabilities) return propCapabilities
    if (!selectedScanWorkflows.length) return []
    const allCaps = new Set<string>()
    void selectedScanWorkflows
    return Array.from(allCaps)
  }, [selectedScanWorkflows, propCapabilities])

  const capabilityStyles = React.useMemo(() => {
    return capabilities.map((capKey) => ({
      key: capKey,
      color: CAPABILITY_CONFIG[capKey]?.color,
    }))
  }, [capabilities])

  return {
    capabilities,
    capabilityStyles,
  }
}

export type ScanConfigEditorState = ReturnType<typeof useScanConfigEditorState>
