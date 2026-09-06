import type { Agent } from "@/types/agent.types"

export type AgentSummary = {
  total: number
  alive: number
  highLoad: number
  offline: number
}

export function buildAgentSummary(nodes: Agent[]): AgentSummary {
  return nodes.reduce(
    (summary, node) => {
      summary.total += 1
      if (node.status === "online") {
        summary.alive += 1
      }
      if (isAgentHighLoad(node)) {
        summary.highLoad += 1
      }
      if (isAgentOffline(node)) {
        summary.offline += 1
      }
      return summary
    },
    { total: 0, alive: 0, highLoad: 0, offline: 0 }
  )
}

export function getAgentLoad(node: Agent): number {
  const heartbeat = node.heartbeat
  if (!heartbeat) {
    return 0
  }

  const taskLoad = node.maxTasks > 0 ? (heartbeat.runningTasks / node.maxTasks) * 100 : 0
  return Math.round(Math.max(heartbeat.cpu, heartbeat.mem, heartbeat.disk, taskLoad))
}

export function getAgentStatus(node: Agent): "highLoad" | "offline" | "normal" {
  if (isAgentOffline(node)) {
    return "offline"
  }
  if (isAgentHighLoad(node)) {
    return "highLoad"
  }
  return "normal"
}

function isAgentHighLoad(node: Agent): boolean {
  const heartbeat = node.heartbeat
  return node.health?.state === "warning" || Boolean(
    heartbeat && (
      heartbeat.cpu >= node.cpuThreshold ||
      heartbeat.mem >= node.memThreshold ||
      heartbeat.disk >= node.diskThreshold
    )
  )
}

function isAgentOffline(node: Agent): boolean {
  return node.status === "offline" || node.health?.state === "error"
}
