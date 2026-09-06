import { useMemo, type CSSProperties } from "react"

import { semanticIcons } from "@/components/icons"
import { getArchitectureFlowRoleIconClassName } from "@/lib/chart-config"

export type RoleKind = "server" | "agent" | "engine"

export type FlowPosition = "left" | "right" | "top" | "bottom"

export type RoleNodeData = {
  title: string
  description: string
  kind: RoleKind
  icon: typeof semanticIcons.concept.server
  iconClassName?: string
}

export type EngineExecutionNodeData = RoleNodeData & {
  id: string
}

export type AgentGroupData = {
  id: string
  label: string
  ip: string
  agent: RoleNodeData
  engineExecutions: EngineExecutionNodeData[]
}

export type ArchitectureFlowData = {
  server: RoleNodeData
  groups: AgentGroupData[]
}

const SOURCE_HANDLE_OFFSET = 6

export const AGENT_COUNT = 3
export const ENGINE_EXECUTIONS_PER_AGENT = [3, 2, 1] as const

export const FLOW_HANDLE_CLASS = "h-2 w-2 border-0 bg-transparent"

export function getSourceHandleStyle(position: FlowPosition): CSSProperties {
  switch (position) {
    case "left":
      return { transform: `translateX(-${SOURCE_HANDLE_OFFSET}px)` }
    case "right":
      return { transform: `translateX(${SOURCE_HANDLE_OFFSET}px)` }
    case "top":
      return { transform: `translateY(-${SOURCE_HANDLE_OFFSET}px)` }
    case "bottom":
      return { transform: `translateY(${SOURCE_HANDLE_OFFSET}px)` }
    default:
      return {}
  }
}

function buildIndexedIds(prefix: string, count: number) {
  return Array.from({ length: count }, (_, index) => `${prefix}-${index + 1}`)
}

function buildEngineExecutionNodes(
  t: (key: string) => string,
  agentIndex: number,
  count: number
): EngineExecutionNodeData[] {
  return buildIndexedIds(`engine-${agentIndex}`, count).map((id, index) => ({
    id,
    title: `${t("flowEngineTitle")} ${agentIndex}-${index + 1}`,
    description: t("flowEngineDesc"),
    kind: "engine",
    icon: semanticIcons.concept.engine,
    iconClassName: getArchitectureFlowRoleIconClassName("engine"),
  }))
}

export function useArchitectureFlowState(t: (key: string) => string): ArchitectureFlowData {
  return useMemo(() => {
    const groups = ENGINE_EXECUTIONS_PER_AGENT.map((engineCount, index) => {
      const agentIndex = index + 1

      return {
        id: `vps-${agentIndex}`,
        label: `${t("flowVpsTitle")} ${agentIndex}`,
        ip: `192.168.1.${agentIndex + 9}`,
        agent: {
          title: `${t("flowAgentTitle")} ${agentIndex}-1`,
          description: t("flowAgentDesc"),
          kind: "agent" as const,
          icon: semanticIcons.concept.agent,
          iconClassName: getArchitectureFlowRoleIconClassName("agent"),
        },
        engineExecutions: buildEngineExecutionNodes(t, agentIndex, engineCount),
      }
    })

    return {
      server: {
        title: t("flowServerTitle"),
        description: t("flowServerDesc"),
        kind: "server",
        icon: semanticIcons.concept.server,
        iconClassName: "text-primary",
      },
      groups,
    }
  }, [t])
}
