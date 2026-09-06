/**
 * Distributed management type definitions (Go backend)
 *
 * Field names mirror the backend `dto.AgentResponse` JSON tags exactly.
 */

export type AgentStatus = 'online' | 'offline' | string
export type AgentLocationState = 'current' | 'expired' | 'unknown'
export type AgentClusterState = 'empty' | 'critical' | 'needsAttention' | 'healthy'
export type AgentClusterReasonCode =
  | 'no_agents'
  | 'no_available_slots'
  | 'offline_agents'
  | 'warning_agents'
  | 'unknown_agents'
  | 'stale_runtime_observations'
  | 'overcommitted_slots'
export type RegistrationTokenState = 'active' | 'expired'

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
  runningTasks: number
  taskSlotsUsed: number
  uptime: number
  agentVersion?: string
  updatedAt: string
  health?: AgentHealth
}

export interface Agent {
  id: number
  name: string
  resourceName?: string
  instanceId?: string
  displayName?: string
  status: AgentStatus
  observedHostname?: string
  connectionIp?: string
  agentVersion?: string
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

export interface AgentLocation {
  latitude: number
  longitude: number
  accuracyRadiusKm: number | null
  sourceObservedIp: string
  providerKey: 'freeipapi'
  resolvedAt: string
}

export interface AgentDetail extends Agent {
  observedSourceIp?: string
  observedIpGeneration: number
  locationState: AgentLocationState
  location: AgentLocation | null
}

export interface AgentClusterExecutionCapacity {
  configuredSlots: number
  occupiedSlots: number
  availableSlots: number
  unavailableSlots: number
  overcommittedSlots: number
}

export interface AgentClusterLocationCoverage {
  positionedCount: number
  unpositionedCount: number
}

export interface AgentClusterSummary {
  resourceName: 'agentClusterSummaries/current'
  generatedAt: string
  executionFreshnessSeconds: 15
  totalNodes: number
  healthyCount: number
  warningCount: number
  offlineCount: number
  unknownCount: number
  staleAgentCount: number
  executionCapacity: AgentClusterExecutionCapacity
  clusterState: AgentClusterState
  reasonCodes: AgentClusterReasonCode[]
  locationCoverage: AgentClusterLocationCoverage
}

export interface AgentLocationMapLocation extends AgentLocation {
  state: Exclude<AgentLocationState, 'unknown'>
}

export interface AgentLocationMapAgent {
  id: number
  resourceName: string
  displayName: string
  status: AgentStatus
  healthState: string
  taskSlotsUsed: number | null
  location: AgentLocationMapLocation
}

export interface ServerLocationMapLocation {
  state: Exclude<AgentLocationState, 'unknown'>
  observedEgressIp: string
  latitude: number
  longitude: number
  accuracyRadiusKm: number | null
  providerKey: 'freeipapi'
  resolvedAt: string
}

export interface AgentLocationMap {
  resourceName: 'agentLocationMaps/current'
  generatedAt: string
  serverLocation: ServerLocationMapLocation | null
  agents: AgentLocationMapAgent[]
}

export interface AgentsResponse {
  results: Agent[]
  totalSize: number
  nextPageToken?: string
  pageSize: number
  total?: number
  page?: number
  totalPages?: number
}

export interface AgentListQueryParams {
  pageSize?: number
  pageToken?: string
  filter?: string
  orderBy?: string
}

export type AgentFilterOptionField = 'status' | 'healthState'

export interface AgentFilterOption {
  value: string
  label: string
  count?: number
}

export interface AgentFilterOptionsResponse {
  results: AgentFilterOption[]
}

export interface UpdateAgentConfigRequest {
  maxTasks?: number
  cpuThreshold?: number
  memThreshold?: number
  diskThreshold?: number
}

export interface RegistrationTokenResponse {
  id: number
  resourceName: string
  token: string
  expiresAt: string
}

export interface RegistrationTokenResource {
  id: number
  resourceName: string
  expiresAt: string
  state: RegistrationTokenState
  agents: Agent[]
}
