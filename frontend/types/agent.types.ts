/**
 * Agent management type definitions (Go backend)
 */

export type AgentStatus = 'online' | 'offline' | string

export interface AgentHealth {
  state: string
  reason?: string
  message?: string
  since?: string | null
}

export interface AgentHeartbeat {
  cpu: number
  mem: number
  disk: number
  tasks: number
  uptime: number
  updatedAt: string
  health?: AgentHealth
}

export interface Agent {
  id: number
  name: string
  status: AgentStatus
  hostname?: string
  ipAddress?: string
  version?: string
  maxTasks: number
  cpuThreshold: number
  memThreshold: number
  diskThreshold: number
  connectedAt?: string | null
  lastHeartbeat?: string | null
  health: AgentHealth
  heartbeat?: AgentHeartbeat
  createdAt: string
}

export interface AgentsResponse {
  results: Agent[]
  total: number
  page: number
  pageSize: number
  totalPages?: number
}

export interface UpdateAgentConfigRequest {
  maxTasks?: number
  cpuThreshold?: number
  memThreshold?: number
  diskThreshold?: number
}

export interface RegistrationTokenResponse {
  token: string
  expiresAt: string
}

