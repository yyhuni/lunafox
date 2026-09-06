import {
  getAgentStatusDistributionStatus,
  type AgentStatusDistributionStatus,
} from "@/lib/agent-status-distribution"
import type { Agent } from "@/types/agent.types"

export const OVERVIEW_AGENT_HEALTH_MATRIX_MAX_TILES = 30
export const OVERVIEW_AGENT_HEALTH_MATRIX_MIN_COLUMNS = 6
export const OVERVIEW_AGENT_HEALTH_MATRIX_MAX_COLUMNS = 10
export const OVERVIEW_AGENT_HEALTH_MATRIX_MAX_ROWS = 3

export type OverviewAgentHealthMatrixColumnCount =
  | 6
  | 7
  | 8
  | 9
  | 10

export type OverviewAgentHealthMatrixNode = {
  id: number
  label: string
  status: AgentStatusDistributionStatus
}

export type OverviewAgentHealthMatrix = {
  nodes: OverviewAgentHealthMatrixNode[]
  hiddenCount: number
  columnCount: OverviewAgentHealthMatrixColumnCount
}

export function getOverviewAgentHealthMatrixColumnCount(
  visibleCellCount: number,
): OverviewAgentHealthMatrixColumnCount {
  return Math.min(
    OVERVIEW_AGENT_HEALTH_MATRIX_MAX_COLUMNS,
    Math.max(
      OVERVIEW_AGENT_HEALTH_MATRIX_MIN_COLUMNS,
      Math.ceil((visibleCellCount * 2) / OVERVIEW_AGENT_HEALTH_MATRIX_MAX_ROWS),
    ),
  ) as OverviewAgentHealthMatrixColumnCount
}

/**
 * The overview keeps node identity for its visible matrix, but caps the card
 * before a large fleet can displace the other first-screen operational signals.
 */
export function buildOverviewAgentHealthMatrix(
  agentNodes: Agent[],
  total = agentNodes.length,
): OverviewAgentHealthMatrix {
  const resolvedTotal = Math.max(total, agentNodes.length)
  const visibleLimit = resolvedTotal > OVERVIEW_AGENT_HEALTH_MATRIX_MAX_TILES
    ? OVERVIEW_AGENT_HEALTH_MATRIX_MAX_TILES - 1
    : OVERVIEW_AGENT_HEALTH_MATRIX_MAX_TILES
  const nodes = agentNodes.slice(0, visibleLimit).map((agentNode) => ({
    id: agentNode.id,
    label: agentNode.displayName ?? agentNode.resourceName ?? agentNode.name,
    status: getAgentStatusDistributionStatus(agentNode),
  }))

  const hiddenCount = Math.max(resolvedTotal - nodes.length, 0)

  return {
    nodes,
    hiddenCount,
    columnCount: getOverviewAgentHealthMatrixColumnCount(nodes.length + (hiddenCount > 0 ? 1 : 0)),
  }
}
