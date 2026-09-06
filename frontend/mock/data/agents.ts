import type {
  Agent,
  AgentFilterOptionField,
  AgentFilterOptionsResponse,
  AgentListQueryParams,
  AgentLocationMapLocation,
  AgentsResponse,
} from '@/types/agent.types'

export const mockAgents: Agent[] = [
  {
    id: 1,
    name: 'agents/1',
    instanceId: 'mock-agent-instance-1',
    displayName: 'edge-01',
    observedHostname: 'edge-01',
    status: 'online',
    connectionIp: '172.20.0.5',
    agentVersion: '0.8.2',
    maxTasks: 5,
    cpuThreshold: 85,
    memThreshold: 85,
    diskThreshold: 90,
    connectedAt: '2024-12-20T08:00:00Z',
    lastHeartbeat: '2024-12-29T10:15:00Z',
    heartbeat: {
      cpu: 22.4,
      mem: 41.8,
      disk: 67.1,
      runningTasks: 2,
      taskSlotsUsed: 2,
      uptime: 86400,
      updatedAt: '2024-12-29T10:15:00Z',
      health: { state: 'healthy' },
    },
    health: { state: 'healthy' },
    createdAt: '2024-12-01T00:00:00Z',
  },
  {
    id: 2,
    name: 'agents/2',
    instanceId: 'mock-agent-instance-2',
    displayName: 'lab-02',
    observedHostname: 'lab-02',
    status: 'online',
    connectionIp: '8.8.8.8',
    agentVersion: '0.8.1',
    maxTasks: 8,
    cpuThreshold: 80,
    memThreshold: 80,
    diskThreshold: 85,
    connectedAt: '2024-12-22T09:40:00Z',
    lastHeartbeat: '2024-12-29T10:10:00Z',
    heartbeat: {
      cpu: 78.2,
      mem: 69.5,
      disk: 52.9,
      runningTasks: 6,
      taskSlotsUsed: 6,
      uptime: 95000,
      updatedAt: '2024-12-29T10:10:00Z',
      health: { state: 'paused', reason: 'cpu', message: 'CPU usage high' },
    },
    health: { state: 'paused', reason: 'cpu', message: 'CPU usage high' },
    createdAt: '2024-12-05T12:30:00Z',
  },
  {
    id: 3,
    name: 'agents/3',
    instanceId: 'mock-agent-instance-3',
    displayName: 'remote-03',
    observedHostname: 'remote-03',
    status: 'offline',
    agentVersion: '0.8.0',
    maxTasks: 4,
    cpuThreshold: 90,
    memThreshold: 90,
    diskThreshold: 95,
    connectedAt: '2024-12-10T07:20:00Z',
    lastHeartbeat: '2024-12-27T18:05:00Z',
    heartbeat: {
      cpu: 0,
      mem: 0,
      disk: 0,
      runningTasks: 0,
      taskSlotsUsed: 0,
      uptime: 0,
      updatedAt: '2024-12-27T18:05:00Z',
      health: { state: 'healthy', reason: 'offline', message: 'Heartbeat lost' },
    },
    health: { state: 'healthy', reason: 'offline', message: 'Heartbeat lost' },
    createdAt: '2024-11-28T15:10:00Z',
  },
  {
    id: 4,
    name: 'agents/4',
    instanceId: 'mock-agent-instance-4',
    displayName: 'prod-04',
    observedHostname: 'prod-04',
    status: 'online',
    connectionIp: '208.67.222.222',
    agentVersion: '0.8.2',
    maxTasks: 10,
    cpuThreshold: 85,
    memThreshold: 85,
    diskThreshold: 90,
    connectedAt: '2024-12-15T10:30:00Z',
    lastHeartbeat: '2024-12-29T10:14:00Z',
    heartbeat: {
      cpu: 45.3,
      mem: 58.2,
      disk: 72.5,
      runningTasks: 5,
      taskSlotsUsed: 5,
      uptime: 120000,
      updatedAt: '2024-12-29T10:14:00Z',
      health: { state: 'healthy' },
    },
    health: { state: 'healthy' },
    createdAt: '2024-12-03T08:20:00Z',
  },
  {
    id: 5,
    name: 'agents/5',
    instanceId: 'mock-agent-instance-5',
    displayName: 'dev-05',
    observedHostname: 'dev-05',
    status: 'online',
    connectionIp: '1.0.0.1',
    agentVersion: '0.8.2',
    maxTasks: 6,
    cpuThreshold: 85,
    memThreshold: 85,
    diskThreshold: 90,
    connectedAt: '2024-12-18T14:20:00Z',
    lastHeartbeat: '2024-12-29T10:13:00Z',
    heartbeat: {
      cpu: 15.8,
      mem: 32.4,
      disk: 48.9,
      runningTasks: 1,
      taskSlotsUsed: 1,
      uptime: 98000,
      updatedAt: '2024-12-29T10:13:00Z',
      health: { state: 'healthy' },
    },
    health: { state: 'healthy' },
    createdAt: '2024-12-08T11:45:00Z',
  },
  {
    id: 6,
    name: 'agents/6',
    instanceId: 'mock-agent-instance-6',
    displayName: 'test-06',
    observedHostname: 'test-06',
    status: 'online',
    connectionIp: '8.8.4.4',
    agentVersion: '0.8.1',
    maxTasks: 5,
    cpuThreshold: 80,
    memThreshold: 80,
    diskThreshold: 85,
    connectedAt: '2024-12-25T09:15:00Z',
    lastHeartbeat: '2024-12-29T10:12:00Z',
    heartbeat: {
      cpu: 88.5,
      mem: 92.1,
      disk: 78.3,
      runningTasks: 4,
      taskSlotsUsed: 4,
      uptime: 35000,
      updatedAt: '2024-12-29T10:12:00Z',
      health: { state: 'paused', reason: 'mem', message: 'Memory usage critical' },
    },
    health: { state: 'paused', reason: 'mem', message: 'Memory usage critical' },
    createdAt: '2024-12-12T16:30:00Z',
  },
  {
    id: 7,
    name: 'agents/7',
    instanceId: 'mock-agent-instance-7',
    displayName: 'backup-07',
    observedHostname: 'backup-07',
    status: 'online',
    connectionIp: '149.112.112.112',
    agentVersion: '0.8.2',
    maxTasks: 8,
    cpuThreshold: 85,
    memThreshold: 85,
    diskThreshold: 90,
    connectedAt: '2024-12-20T11:00:00Z',
    lastHeartbeat: '2024-12-29T10:11:00Z',
    heartbeat: {
      cpu: 35.7,
      mem: 48.9,
      disk: 85.2,
      runningTasks: 3,
      taskSlotsUsed: 3,
      uptime: 78000,
      updatedAt: '2024-12-29T10:11:00Z',
      health: { state: 'healthy' },
    },
    health: { state: 'healthy' },
    createdAt: '2024-12-06T13:20:00Z',
  },
  {
    id: 8,
    name: 'agents/8',
    instanceId: 'mock-agent-instance-8',
    displayName: 'cloud-08',
    observedHostname: 'cloud-08',
    status: 'online',
    connectionIp: '208.67.220.220',
    agentVersion: '0.8.2',
    maxTasks: 12,
    cpuThreshold: 85,
    memThreshold: 85,
    diskThreshold: 90,
    connectedAt: '2024-12-22T15:45:00Z',
    lastHeartbeat: '2024-12-29T10:10:00Z',
    heartbeat: {
      cpu: 52.3,
      mem: 64.7,
      disk: 55.8,
      runningTasks: 8,
      taskSlotsUsed: 8,
      uptime: 65000,
      updatedAt: '2024-12-29T10:10:00Z',
      health: { state: 'healthy' },
    },
    health: { state: 'healthy' },
    createdAt: '2024-12-09T09:30:00Z',
  },
  {
    id: 9,
    name: 'agents/9',
    instanceId: 'mock-agent-instance-9',
    displayName: 'staging-09',
    observedHostname: 'staging-09',
    status: 'offline',
    agentVersion: '0.8.0',
    maxTasks: 5,
    cpuThreshold: 90,
    memThreshold: 90,
    diskThreshold: 95,
    connectedAt: '2024-12-12T08:30:00Z',
    lastHeartbeat: '2024-12-28T20:15:00Z',
    heartbeat: {
      cpu: 0,
      mem: 0,
      disk: 0,
      runningTasks: 0,
      taskSlotsUsed: 0,
      uptime: 0,
      updatedAt: '2024-12-28T20:15:00Z',
      health: { state: 'healthy', reason: 'offline', message: 'Connection lost' },
    },
    health: { state: 'healthy', reason: 'offline', message: 'Connection lost' },
    createdAt: '2024-11-30T14:50:00Z',
  },
  {
    id: 10,
    name: 'agents/10',
    instanceId: 'mock-agent-instance-10',
    displayName: 'monitor-10',
    observedHostname: 'monitor-10',
    status: 'online',
    agentVersion: '0.8.2',
    maxTasks: 7,
    cpuThreshold: 85,
    memThreshold: 85,
    diskThreshold: 90,
    connectedAt: '2024-12-24T12:00:00Z',
    lastHeartbeat: '2024-12-29T10:09:00Z',
    heartbeat: {
      cpu: 28.4,
      mem: 39.6,
      disk: 62.3,
      runningTasks: 2,
      taskSlotsUsed: 2,
      uptime: 42000,
      updatedAt: '2024-12-29T10:09:00Z',
      health: { state: 'healthy' },
    },
    health: { state: 'healthy' },
    createdAt: '2024-12-11T10:15:00Z',
  },
]

const MOCK_AGENT_OPERATIONAL_GENERATED_AT = '2026-08-04T07:00:00Z'
const MOCK_REGISTRATION_TOKEN_NAME = 'agentRegistrationTokens/101'

const mockAgentLocationsById: Partial<Record<number, AgentLocationMapLocation>> = {
  1: {
    state: 'current',
    latitude: 39.9042,
    longitude: 116.4074,
    accuracyRadiusKm: null,
    sourceObservedIp: '1.1.1.1',
    providerKey: 'freeipapi',
    resolvedAt: '2026-08-02T07:00:00Z',
  },
  2: {
    state: 'expired',
    latitude: 35.6762,
    longitude: 139.6503,
    accuracyRadiusKm: 25,
    sourceObservedIp: '8.8.8.8',
    providerKey: 'freeipapi',
    resolvedAt: '2026-07-01T07:00:00Z',
  },
  3: {
    state: 'current',
    latitude: 40.7128,
    longitude: -74.006,
    accuracyRadiusKm: null,
    sourceObservedIp: '9.9.9.9',
    providerKey: 'freeipapi',
    resolvedAt: '2026-08-01T07:00:00Z',
  },
  4: {
    state: 'current',
    latitude: 51.5072,
    longitude: -0.1276,
    accuracyRadiusKm: 18,
    sourceObservedIp: '208.67.222.222',
    providerKey: 'freeipapi',
    resolvedAt: '2026-08-01T08:00:00Z',
  },
  5: {
    state: 'current',
    latitude: 37.7595,
    longitude: -122.4367,
    accuracyRadiusKm: null,
    sourceObservedIp: '1.0.0.1',
    providerKey: 'freeipapi',
    resolvedAt: '2026-08-01T09:00:00Z',
  },
  6: {
    state: 'current',
    latitude: 19.076,
    longitude: 72.8777,
    accuracyRadiusKm: 30,
    sourceObservedIp: '8.8.4.4',
    providerKey: 'freeipapi',
    resolvedAt: '2026-08-01T10:00:00Z',
  },
  7: {
    state: 'current',
    latitude: 1.3521,
    longitude: 103.8198,
    accuracyRadiusKm: null,
    sourceObservedIp: '149.112.112.112',
    providerKey: 'freeipapi',
    resolvedAt: '2026-08-01T11:00:00Z',
  },
  8: {
    state: 'current',
    latitude: 50.1109,
    longitude: 8.6821,
    accuracyRadiusKm: 20,
    sourceObservedIp: '208.67.220.220',
    providerKey: 'freeipapi',
    resolvedAt: '2026-08-01T12:00:00Z',
  },
}

export function getMockAgentClusterSummary() {
  return {
    name: 'agentClusterSummaries/current',
    generatedAt: MOCK_AGENT_OPERATIONAL_GENERATED_AT,
    executionFreshnessSeconds: 15,
    totalNodes: 10,
    healthyCount: 6,
    warningCount: 2,
    offlineCount: 2,
    unknownCount: 0,
    staleAgentCount: 0,
    executionCapacity: {
      configuredSlots: 70,
      occupiedSlots: 21,
      availableSlots: 27,
      unavailableSlots: 22,
      overcommittedSlots: 0,
    },
    clusterState: 'needsAttention',
    reasonCodes: ['offline_agents', 'warning_agents'],
    locationCoverage: {
      positionedCount: 8,
      unpositionedCount: 2,
    },
  }
}

export function getMockAgentLocationMap() {
  const agents = mockAgents.flatMap((agent) => {
    const location = mockAgentLocationsById[agent.id]
    if (!location) {
      return []
    }
    return [{
      name: agent.name,
      displayName: agent.displayName,
      status: agent.status,
      healthState: agent.health.state,
      taskSlotsUsed: agent.heartbeat?.taskSlotsUsed ?? null,
      location,
    }]
  })

  return {
    name: 'agentLocationMaps/current',
    generatedAt: MOCK_AGENT_OPERATIONAL_GENERATED_AT,
    serverLocation: {
      state: 'current',
      observedEgressIp: '4.2.2.2',
      latitude: 31.2304,
      longitude: 121.4737,
      accuracyRadiusKm: null,
      providerKey: 'freeipapi',
      resolvedAt: '2026-08-02T06:00:00Z',
    },
    agents,
  }
}

function parseMockAgentPageToken(value: string | undefined) {
  const match = value?.match(/^mock-agent-page-(\d+)$/)
  return match ? Number.parseInt(match[1], 10) : 1
}

function extractFilterValues(filter: string | undefined, field: string) {
  return Array.from(String(filter ?? "").matchAll(new RegExp(`${field}(?:==|=)\"((?:\\\\.|[^\"\\\\])*)\"`, "g")))
    .map((match) => match[1]?.replace(/\\"/g, '"').replace(/\\\\/g, "\\") ?? "")
    .filter(Boolean)
}

function matchesAgentFilter(agent: Agent, filter: string | undefined) {
  if (!filter) {
    return true
  }
  const searchValues = [
    ...extractFilterValues(filter, "displayName"),
    ...extractFilterValues(filter, "observedHostname"),
    ...extractFilterValues(filter, "connectionIp"),
  ]
  const statusValues = extractFilterValues(filter, "status")
  const healthValues = extractFilterValues(filter, "healthState")
  const searchable = [agent.displayName, agent.name, agent.observedHostname, agent.connectionIp]
    .filter(Boolean)
    .join(" ")
    .toLowerCase()
  const searchMatched = searchValues.length === 0 || searchValues.some((value) => searchable.includes(value.toLowerCase()))
  const statusMatched = (statusValues.length === 0 || statusValues.includes(agent.status)) &&
    (!filter.includes('status!="online"') || agent.status !== "online") &&
    (!filter.includes('status!="offline"') || agent.status !== "offline")
  const healthState = agent.health?.state ?? agent.heartbeat?.health?.state ?? ""
  const healthMatched = healthValues.length === 0 || healthValues.includes(healthState)

  return searchMatched && statusMatched && healthMatched
}

function sortMockAgents(agents: Agent[], orderBy: string | undefined) {
  const direction = orderBy === "createdAt" || orderBy === "createdAt asc" ? 1 : -1
  return [...agents].sort((left, right) => {
    const diff = new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime()
    if (diff !== 0) {
      return diff * direction
    }
    return (left.id - right.id) * direction
  })
}

export function getMockAgents(params: AgentListQueryParams = {}): AgentsResponse {
  const pageSize = params.pageSize ?? 10
  const page = parseMockAgentPageToken(params.pageToken)
  const filtered = sortMockAgents(mockAgents.filter((agent) => matchesAgentFilter(agent, params.filter)), params.orderBy)
  const total = filtered.length
  const totalPages = Math.ceil(total / pageSize) || 1
  const start = (page - 1) * pageSize
  const results = filtered.slice(start, start + pageSize)
  const nextPage = page < totalPages ? page + 1 : undefined

  return {
    results,
    totalSize: total,
    total,
    page,
    pageSize,
    totalPages,
    nextPageToken: nextPage ? `mock-agent-page-${nextPage}` : undefined,
  }
}

export function getMockAgentFilterOptions(field: AgentFilterOptionField): AgentFilterOptionsResponse {
  const counts = new Map<string, number>()
  for (const agent of mockAgents) {
    const value = field === "status" ? agent.status : agent.health?.state
    if (!value) {
      continue
    }
    counts.set(value, (counts.get(value) ?? 0) + 1)
  }
  return {
    results: Array.from(counts.entries())
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([value, count]) => ({ value, label: value, count })),
  }
}

export function getMockAgentById(id: number) {
  const agent = mockAgents.find((candidate) => candidate.id === id)
  if (!agent) {
    return undefined
  }
  const positionedLocation = mockAgentLocationsById[id]
  if (!positionedLocation) {
    return {
      ...agent,
      observedIpGeneration: 0,
      locationState: 'unknown',
      location: null,
    }
  }
  const { state, ...location } = positionedLocation
  return {
    ...agent,
    observedSourceIp: positionedLocation.sourceObservedIp,
    observedIpGeneration: 1,
    locationState: state,
    location,
  }
}

export function deleteMockAgent(id: number): boolean {
  const index = mockAgents.findIndex((agent) => agent.id === id)
  if (index === -1) {
    return false
  }

  mockAgents.splice(index, 1)
  return true
}

export function getMockRegistrationToken() {
  return {
    name: MOCK_REGISTRATION_TOKEN_NAME,
    token: 'a1b2c3d4',
    expiresAt: '2099-08-04T08:00:00Z',
  }
}

export function getMockRegistrationTokenById(id: number) {
  if (id !== 101) {
    return undefined
  }
  return {
    name: MOCK_REGISTRATION_TOKEN_NAME,
    expiresAt: '2099-08-04T08:00:00Z',
    state: 'active',
    agents: mockAgents.slice(0, 2),
  }
}
