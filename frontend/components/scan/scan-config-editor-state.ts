import React from "react"

import { CAPABILITY_CONFIG, parseWorkflowCapabilities } from "@/lib/workflow-config"

import type { ScanWorkflow } from "@/types/workflow.types"

interface ScanConfigEditorStateOptions {
  selectedWorkflows?: ScanWorkflow[]
  selectedCapabilities?: string[]
}

export function useScanConfigEditorState({
  selectedWorkflows = [],
  selectedCapabilities: propCapabilities,
}: ScanConfigEditorStateOptions) {
  const capabilities = React.useMemo(() => {
    if (propCapabilities) return propCapabilities
    if (!selectedWorkflows.length) return []
    const allCaps = new Set<string>()
    selectedWorkflows.forEach((workflow) => {
      parseWorkflowCapabilities(workflow.configuration || "").forEach((cap) => allCaps.add(cap))
    })
    return Array.from(allCaps)
  }, [selectedWorkflows, propCapabilities])

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
