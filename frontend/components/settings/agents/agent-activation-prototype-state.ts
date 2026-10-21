import type { AgentClusterSummary } from "@/types/agent.types"

// Deliberately recognizable fixtures: these keys are not a production license format.
export const ACTIVATION_SAMPLE_KEYS = ["LF-DEMO-1M", "LF-DEMO-2M", "LF-DEMO-3M"] as const
export const PROTOTYPE_TODAY = "2026-10-18"

export type ActivationPreviewLicense = {
  months: number
  expiresAt: string
  licenseId: string
}

export type ActivationPreviewNode = {
  id: number
  name: string
  address: string
  cpu: number
  memory: number
}

export const INITIAL_PREVIEW_NODES: ActivationPreviewNode[] = [
  { id: 1, name: "默认 Agent", address: "10.0.0.2", cpu: 18, memory: 32 },
  { id: 2, name: "华东扫描节点", address: "10.0.0.11", cpu: 24, memory: 46 },
  { id: 3, name: "华南扫描节点", address: "10.0.0.12", cpu: 12, memory: 28 },
]

export function sampleLicense(key: string): ActivationPreviewLicense | null {
  const index = ACTIVATION_SAMPLE_KEYS.indexOf(key.trim().toUpperCase() as typeof ACTIVATION_SAMPLE_KEYS[number])
  if (index < 0) return null
  const months = index + 1
  const date = new Date(`${PROTOTYPE_TODAY}T00:00:00Z`)
  date.setUTCMonth(date.getUTCMonth() + months)
  return {
    months,
    expiresAt: date.toISOString().slice(0, 10),
    licenseId: `LF-DEMO-${months}M-2026`,
  }
}

export function previewClusterSummary(nodeCount: number, agentLimit: number): AgentClusterSummary {
  return {
    resourceName: "agentClusterSummaries/current",
    generatedAt: `${PROTOTYPE_TODAY}T00:00:00Z`,
    executionFreshnessSeconds: 15,
    agentLimit,
    totalNodes: nodeCount,
    healthyCount: nodeCount,
    warningCount: 0,
    offlineCount: 0,
    unknownCount: 0,
    staleAgentCount: 0,
    executionCapacity: {
      configuredSlots: nodeCount * 5,
      occupiedSlots: 0,
      availableSlots: nodeCount * 5,
      unavailableSlots: 0,
      overcommittedSlots: 0,
    },
    clusterState: "healthy",
    reasonCodes: [],
    locationCoverage: { positionedCount: nodeCount, unpositionedCount: 0 },
  }
}
