import React from "react"

import { CAPABILITY_CONFIG, parseEngineCapabilities } from "@/lib/engine-config"

import type { ScanEngine } from "@/types/engine.types"

interface ScanConfigEditorStateOptions {
  selectedEngines?: ScanEngine[]
  selectedCapabilities?: string[]
}

export function useScanConfigEditorState({
  selectedEngines = [],
  selectedCapabilities: propCapabilities,
}: ScanConfigEditorStateOptions) {
  const capabilities = React.useMemo(() => {
    if (propCapabilities) return propCapabilities
    if (!selectedEngines.length) return []
    const allCaps = new Set<string>()
    selectedEngines.forEach((engine) => {
      parseEngineCapabilities(engine.configuration || "").forEach((cap) => allCaps.add(cap))
    })
    return Array.from(allCaps)
  }, [selectedEngines, propCapabilities])

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
