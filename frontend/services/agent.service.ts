/**
 * Distributed management API service (Go backend)
 */

import type { AxiosError } from 'axios'
import { api } from '@/lib/api-client'
import { parseAgentName, parseAgentRegistrationTokenName } from '@/lib/resource-name'
import type {
  Agent,
  AgentClusterReasonCode,
  AgentClusterState,
  AgentClusterSummary,
  AgentDetail,
  AgentFilterOptionField,
  AgentFilterOptionsResponse,
  AgentListQueryParams,
  AgentLocation,
  AgentLocationMap,
  AgentLocationMapLocation,
  AgentsResponse,
  RegistrationTokenResponse,
  RegistrationTokenResource,
  RegistrationTokenState,
  ServerLocationMapLocation,
  UpdateAgentConfigRequest,
} from '@/types/agent.types'
import type { AgentLogItem, AgentLogsResponse } from '@/types/agent-log.types'

const BASE_URL = '/admin/agents'
const CLUSTER_SUMMARY_URL = '/admin/agentClusterSummaries/current'
const LOCATION_MAP_URL = '/admin/agentLocationMaps/current'
const REGISTRATION_TOKENS_URL = '/admin/agentRegistrationTokens'
const CLUSTER_SUMMARY_NAME = 'agentClusterSummaries/current' as const
const LOCATION_MAP_NAME = 'agentLocationMaps/current' as const

const DEFAULT_LOG_LIMIT = 200
const MAX_LOG_LIMIT = 500

export interface FetchAgentLogsParams {
  agentNodeId: number
  container: string
  limit?: number
  cursor?: string
  direction?: 'newer' | 'older'
  signal?: AbortSignal
}

export class AgentLogQueryError extends Error {
  code: string
  status?: number

  constructor(code: string, message: string, status?: number) {
    super(message)
    this.name = 'AgentLogQueryError'
    this.code = code
    this.status = status
  }
}

function buildAgentListParams(params?: AgentListQueryParams) {
  return {
    pageSize: params?.pageSize ?? 10,
    ...(params?.pageToken && { pageToken: params.pageToken }),
    ...(params?.filter && { filter: params.filter }),
    ...(params?.orderBy && { orderBy: params.orderBy }),
  }
}

function normalizeAgentsResponse(response: AgentsResponse, params?: AgentListQueryParams): AgentsResponse {
  const total = response.total ?? response.totalSize ?? response.results.length
  const normalizedPage = response.page ?? 1
  const normalizedPageSize = response.pageSize ?? params?.pageSize ?? 10
  const totalPages = response.totalPages ?? Math.ceil(total / normalizedPageSize)

  return {
    ...response,
    results: response.results.map((agent, index) => normalizeAgent(agent, `results[${index}]`)),
    totalSize: response.totalSize ?? total,
    total,
    page: normalizedPage,
    pageSize: normalizedPageSize,
    totalPages,
  }
}

type JsonObject = Record<string, unknown>

function invalidAgentResponse(path: string): never {
  throw new Error(`Agent API response is invalid at ${path}`)
}

function requireObject(value: unknown, path: string): JsonObject {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return invalidAgentResponse(path)
  }
  return value as JsonObject
}

function requireString(value: unknown, path: string): string {
  if (typeof value !== 'string' || value.trim() === '') {
    return invalidAgentResponse(path)
  }
  return value
}

function requireRawString(value: unknown, path: string): string {
  if (typeof value !== 'string') {
    return invalidAgentResponse(path)
  }
  return value
}

function requireTimestamp(value: unknown, path: string): string {
  const timestamp = requireString(value, path)
  if (!Number.isFinite(Date.parse(timestamp))) {
    return invalidAgentResponse(path)
  }
  return timestamp
}

function requireInteger(value: unknown, path: string, minimum = 0): number {
  if (!Number.isSafeInteger(value) || (value as number) < minimum) {
    return invalidAgentResponse(path)
  }
  return value as number
}

function requireFiniteNumber(value: unknown, path: string, minimum: number, maximum: number): number {
  if (typeof value !== 'number' || !Number.isFinite(value) || value < minimum || value > maximum) {
    return invalidAgentResponse(path)
  }
  return value
}

function requireNullableRadius(value: unknown, path: string): number | null {
  if (value === null) {
    return null
  }
  return requireFiniteNumber(value, path, 0, Number.MAX_VALUE)
}

function requireEnum<T extends string>(value: unknown, allowed: ReadonlySet<string>, path: string): T {
  if (typeof value !== 'string' || !allowed.has(value)) {
    return invalidAgentResponse(path)
  }
  return value as T
}

function parseCanonicalAgentName(value: unknown, path: string): { id: number; resourceName: string } {
  const resourceName = requireString(value, path)
  try {
    return { id: parseAgentName(resourceName), resourceName }
  } catch {
    return invalidAgentResponse(path)
  }
}

function parseCanonicalRegistrationTokenName(value: unknown, path: string): { id: number; resourceName: string } {
  const resourceName = requireString(value, path)
  try {
    return { id: parseAgentRegistrationTokenName(resourceName), resourceName }
  } catch {
    return invalidAgentResponse(path)
  }
}

function normalizeAgent(value: unknown, path = 'agent'): Agent {
  const record = requireObject(value, path)
  if ('ipAddress' in record) {
    return invalidAgentResponse(`${path}.ipAddress`)
  }
  const { connectionIp: rawConnectionIp, ...agentRecord } = record
  const id = requireInteger(record.id, `${path}.id`, 1)
  const canonical = parseCanonicalAgentName(record.name, `${path}.name`)
  if (canonical.id !== id) {
    return invalidAgentResponse(`${path}.name`)
  }
  const displayName = requireString(record.displayName, `${path}.displayName`)
  const health = requireObject(record.health, `${path}.health`)
  // Registration precedes the first heartbeat, so the raw runtime health state can be empty.
  const healthState = requireRawString(health.state, `${path}.health.state`)
  const connectionIp = rawConnectionIp === undefined
    ? undefined
    : requireString(rawConnectionIp, `${path}.connectionIp`)

  return {
    ...(agentRecord as unknown as Agent),
    id,
    name: displayName,
    displayName,
    resourceName: canonical.resourceName,
    status: requireString(record.status, `${path}.status`),
    maxTasks: requireInteger(record.maxTasks, `${path}.maxTasks`, 1),
    cpuThreshold: requireInteger(record.cpuThreshold, `${path}.cpuThreshold`, 1),
    memThreshold: requireInteger(record.memThreshold, `${path}.memThreshold`, 1),
    diskThreshold: requireInteger(record.diskThreshold, `${path}.diskThreshold`, 1),
    health: {
      ...(health as unknown as Agent['health']),
      state: healthState,
    },
    ...(connectionIp ? { connectionIp } : {}),
    createdAt: requireTimestamp(record.createdAt, `${path}.createdAt`),
  }
}

const LOCATION_STATES = new Set(['current', 'expired'])
const CLUSTER_STATES = new Set<AgentClusterState>(['empty', 'critical', 'needsAttention', 'healthy'])
const CLUSTER_REASON_CODES = new Set<AgentClusterReasonCode>([
  'no_agents',
  'no_available_slots',
  'offline_agents',
  'warning_agents',
  'unknown_agents',
  'stale_runtime_observations',
  'overcommitted_slots',
])
const REGISTRATION_TOKEN_STATES = new Set<RegistrationTokenState>(['active', 'expired'])

function parseAgentLocation(value: unknown, path: string): AgentLocation {
  const record = requireObject(value, path)
  return {
    latitude: requireFiniteNumber(record.latitude, `${path}.latitude`, -90, 90),
    longitude: requireFiniteNumber(record.longitude, `${path}.longitude`, -180, 180),
    accuracyRadiusKm: requireNullableRadius(record.accuracyRadiusKm, `${path}.accuracyRadiusKm`),
    sourceObservedIp: requireString(record.sourceObservedIp, `${path}.sourceObservedIp`),
    providerKey: requireEnum(record.providerKey, new Set(['freeipapi']), `${path}.providerKey`),
    resolvedAt: requireTimestamp(record.resolvedAt, `${path}.resolvedAt`),
  }
}

function normalizeAgentDetail(value: unknown, expectedId: number): AgentDetail {
  const record = requireObject(value, 'agent')
  const agent = normalizeAgent(record)
  if (agent.id !== expectedId) {
    return invalidAgentResponse('agent.name')
  }

  const locationState = requireEnum<AgentDetail['locationState']>(
    record.locationState,
    new Set(['current', 'expired', 'unknown']),
    'agent.locationState',
  )
  const location = record.location === null ? null : parseAgentLocation(record.location, 'agent.location')
  if ((locationState === 'unknown') !== (location === null)) {
    return invalidAgentResponse('agent.location')
  }

  const observedSourceIp = record.observedSourceIp === undefined
    ? undefined
    : requireString(record.observedSourceIp, 'agent.observedSourceIp')

  return {
    ...agent,
    ...(observedSourceIp ? { observedSourceIp } : {}),
    observedIpGeneration: requireInteger(record.observedIpGeneration, 'agent.observedIpGeneration'),
    locationState,
    location,
  }
}

function parseAgentClusterSummary(value: unknown): AgentClusterSummary {
  const record = requireObject(value, 'agentClusterSummary')
  if (record.name !== CLUSTER_SUMMARY_NAME) {
    return invalidAgentResponse('agentClusterSummary.name')
  }
  const totalNodes = requireInteger(record.totalNodes, 'agentClusterSummary.totalNodes')
  const healthyCount = requireInteger(record.healthyCount, 'agentClusterSummary.healthyCount')
  const warningCount = requireInteger(record.warningCount, 'agentClusterSummary.warningCount')
  const offlineCount = requireInteger(record.offlineCount, 'agentClusterSummary.offlineCount')
  const unknownCount = requireInteger(record.unknownCount, 'agentClusterSummary.unknownCount')
  if (healthyCount + warningCount + offlineCount + unknownCount !== totalNodes) {
    return invalidAgentResponse('agentClusterSummary.nodeCounts')
  }

  const capacity = requireObject(record.executionCapacity, 'agentClusterSummary.executionCapacity')
  const configuredSlots = requireInteger(capacity.configuredSlots, 'agentClusterSummary.executionCapacity.configuredSlots')
  const occupiedSlots = requireInteger(capacity.occupiedSlots, 'agentClusterSummary.executionCapacity.occupiedSlots')
  const availableSlots = requireInteger(capacity.availableSlots, 'agentClusterSummary.executionCapacity.availableSlots')
  const unavailableSlots = requireInteger(capacity.unavailableSlots, 'agentClusterSummary.executionCapacity.unavailableSlots')
  if (configuredSlots !== occupiedSlots + availableSlots + unavailableSlots) {
    return invalidAgentResponse('agentClusterSummary.executionCapacity')
  }

  const coverage = requireObject(record.locationCoverage, 'agentClusterSummary.locationCoverage')
  const positionedCount = requireInteger(coverage.positionedCount, 'agentClusterSummary.locationCoverage.positionedCount')
  const unpositionedCount = requireInteger(coverage.unpositionedCount, 'agentClusterSummary.locationCoverage.unpositionedCount')
  if (positionedCount + unpositionedCount !== totalNodes) {
    return invalidAgentResponse('agentClusterSummary.locationCoverage')
  }
  if (!Array.isArray(record.reasonCodes)) {
    return invalidAgentResponse('agentClusterSummary.reasonCodes')
  }
  const reasonCodes = record.reasonCodes.map((reasonCode, index) =>
    requireEnum<AgentClusterReasonCode>(reasonCode, CLUSTER_REASON_CODES, `agentClusterSummary.reasonCodes[${index}]`))
  if (new Set(reasonCodes).size !== reasonCodes.length) {
    return invalidAgentResponse('agentClusterSummary.reasonCodes')
  }
  const executionFreshnessSeconds = requireInteger(
    record.executionFreshnessSeconds,
    'agentClusterSummary.executionFreshnessSeconds',
    1,
  )
  if (executionFreshnessSeconds !== 15) {
    return invalidAgentResponse('agentClusterSummary.executionFreshnessSeconds')
  }

  return {
    resourceName: CLUSTER_SUMMARY_NAME,
    generatedAt: requireTimestamp(record.generatedAt, 'agentClusterSummary.generatedAt'),
    executionFreshnessSeconds: 15,
    totalNodes,
    healthyCount,
    warningCount,
    offlineCount,
    unknownCount,
    staleAgentCount: requireInteger(record.staleAgentCount, 'agentClusterSummary.staleAgentCount'),
    executionCapacity: {
      configuredSlots,
      occupiedSlots,
      availableSlots,
      unavailableSlots,
      overcommittedSlots: requireInteger(
        capacity.overcommittedSlots,
        'agentClusterSummary.executionCapacity.overcommittedSlots',
      ),
    },
    clusterState: requireEnum(record.clusterState, CLUSTER_STATES, 'agentClusterSummary.clusterState'),
    reasonCodes,
    locationCoverage: { positionedCount, unpositionedCount },
  }
}

function parseMapLocation(value: unknown, path: string): AgentLocationMapLocation {
  const record = requireObject(value, path)
  return {
    state: requireEnum(record.state, LOCATION_STATES, `${path}.state`),
    ...parseAgentLocation(record, path),
  }
}

function parseServerMapLocation(value: unknown, path: string): ServerLocationMapLocation {
  const record = requireObject(value, path)
  return {
    state: requireEnum(record.state, LOCATION_STATES, `${path}.state`),
    observedEgressIp: requireString(record.observedEgressIp, `${path}.observedEgressIp`),
    latitude: requireFiniteNumber(record.latitude, `${path}.latitude`, -90, 90),
    longitude: requireFiniteNumber(record.longitude, `${path}.longitude`, -180, 180),
    accuracyRadiusKm: requireNullableRadius(record.accuracyRadiusKm, `${path}.accuracyRadiusKm`),
    providerKey: requireEnum(record.providerKey, new Set(['freeipapi']), `${path}.providerKey`),
    resolvedAt: requireTimestamp(record.resolvedAt, `${path}.resolvedAt`),
  }
}

function parseAgentLocationMap(value: unknown): AgentLocationMap {
  const record = requireObject(value, 'agentLocationMap')
  if (record.name !== LOCATION_MAP_NAME) {
    return invalidAgentResponse('agentLocationMap.name')
  }
  if (!Array.isArray(record.agents)) {
    return invalidAgentResponse('agentLocationMap.agents')
  }
  const agents = record.agents.map((value, index) => {
    const path = `agentLocationMap.agents[${index}]`
    const agent = requireObject(value, path)
    const canonical = parseCanonicalAgentName(agent.name, `${path}.name`)
    const taskSlotsUsed = agent.taskSlotsUsed === null
      ? null
      : requireInteger(agent.taskSlotsUsed, `${path}.taskSlotsUsed`)
    return {
      id: canonical.id,
      resourceName: canonical.resourceName,
      displayName: requireString(agent.displayName, `${path}.displayName`),
      status: requireString(agent.status, `${path}.status`),
      healthState: requireString(agent.healthState, `${path}.healthState`),
      taskSlotsUsed,
      location: parseMapLocation(agent.location, `${path}.location`),
    }
  })
  if (new Set(agents.map((agent) => agent.resourceName)).size !== agents.length) {
    return invalidAgentResponse('agentLocationMap.agents')
  }

  return {
    resourceName: LOCATION_MAP_NAME,
    generatedAt: requireTimestamp(record.generatedAt, 'agentLocationMap.generatedAt'),
    serverLocation: record.serverLocation === null
      ? null
      : parseServerMapLocation(record.serverLocation, 'agentLocationMap.serverLocation'),
    agents,
  }
}

function parseRegistrationTokenResponse(value: unknown): RegistrationTokenResponse {
  const record = requireObject(value, 'registrationToken')
  const canonical = parseCanonicalRegistrationTokenName(record.name, 'registrationToken.name')
  return {
    id: canonical.id,
    resourceName: canonical.resourceName,
    token: requireString(record.token, 'registrationToken.token'),
    expiresAt: requireTimestamp(record.expiresAt, 'registrationToken.expiresAt'),
  }
}

function parseRegistrationTokenResource(value: unknown, expectedName: string): RegistrationTokenResource {
  const record = requireObject(value, 'registrationToken')
  if ('token' in record) {
    return invalidAgentResponse('registrationToken.token')
  }
  const canonical = parseCanonicalRegistrationTokenName(record.name, 'registrationToken.name')
  if (canonical.resourceName !== expectedName) {
    return invalidAgentResponse('registrationToken.name')
  }
  if (!Array.isArray(record.agents)) {
    return invalidAgentResponse('registrationToken.agents')
  }
  return {
    id: canonical.id,
    resourceName: canonical.resourceName,
    expiresAt: requireTimestamp(record.expiresAt, 'registrationToken.expiresAt'),
    state: requireEnum(record.state, REGISTRATION_TOKEN_STATES, 'registrationToken.state'),
    agents: record.agents.map((agent, index) => normalizeAgent(agent, `registrationToken.agents[${index}]`)),
  }
}

export const agentService = {
  async getAgents(params?: AgentListQueryParams, signal?: AbortSignal): Promise<AgentsResponse> {
    const response = await api.get<AgentsResponse>(BASE_URL, {
      params: buildAgentListParams(params),
      ...(signal ? { signal } : {}),
    })
    return normalizeAgentsResponse(response.data, params)
  },

  async getAgentFilterOptions(field: AgentFilterOptionField): Promise<AgentFilterOptionsResponse> {
    const response = await api.get<AgentFilterOptionsResponse>(`${BASE_URL}/filterOptions`, { params: { field } })
    return response.data
  },

  async getAgent(resourceName: string, signal?: AbortSignal): Promise<AgentDetail> {
    const id = parseAgentRegistrationSafe(resourceName)
    const response = await api.get<unknown>(`${BASE_URL}/${id}`, signal ? { signal } : undefined)
    return normalizeAgentDetail(response.data, id)
  },

  async getAgentClusterSummary(signal?: AbortSignal): Promise<AgentClusterSummary> {
    const response = await api.get<unknown>(CLUSTER_SUMMARY_URL, signal ? { signal } : undefined)
    return parseAgentClusterSummary(response.data)
  },

  async getAgentLocationMap(signal?: AbortSignal): Promise<AgentLocationMap> {
    const response = await api.get<unknown>(LOCATION_MAP_URL, signal ? { signal } : undefined)
    return parseAgentLocationMap(response.data)
  },

  async deleteAgent(id: number): Promise<void> {
    await api.delete(`${BASE_URL}/${id}`)
  },

  async updateDistributedConfig(id: number, data: UpdateAgentConfigRequest): Promise<Agent> {
    const updateMask = Object.keys(data).join(',')
    const body = {
      name: `agents/${id}`,
      updateMask,
      ...data,
    }
    const response = await api.patch<Agent>(`${BASE_URL}/${id}`, body)
    return normalizeAgent(response.data)
  },

  async createRegistrationToken(): Promise<RegistrationTokenResponse> {
    const response = await api.post<unknown>(REGISTRATION_TOKENS_URL)
    return parseRegistrationTokenResponse(response.data)
  },

  async getRegistrationToken(resourceName: string, signal?: AbortSignal): Promise<RegistrationTokenResource> {
    const id = parseRegistrationTokenSafe(resourceName)
    const response = await api.get<unknown>(
      `${REGISTRATION_TOKENS_URL}/${id}`,
      signal ? { signal } : undefined,
    )
    return parseRegistrationTokenResource(response.data, resourceName)
  },

  async fetchDistributedLogs(params: FetchAgentLogsParams): Promise<AgentLogsResponse> {
    const container = params.container.trim()
    if (!container) {
      throw new AgentLogQueryError('bad_request', 'Container is required')
    }

    const limit = requireLogLimit(params.limit)
    const cursor = params.cursor?.trim() ?? ''
    const direction = params.direction?.trim() ?? ''
    const queryParams: Record<string, string> = {
      container,
      pageSize: String(limit),
    }
    if (cursor) {
      queryParams.pageToken = cursor
    }
    if (direction) {
      queryParams.direction = direction
    }

    let payload: {
      results?: unknown[]
      nextPageToken?: string
      previousPageToken?: string
      hasOlder?: unknown
      hasNewer?: unknown
      caughtUp?: unknown
      gap?: unknown
      gapReason?: unknown
    }
    try {
      const response = await api.get<{
        results?: unknown[]
        nextPageToken?: string
        previousPageToken?: string
        hasOlder?: unknown
        hasNewer?: unknown
        caughtUp?: unknown
        gap?: unknown
        gapReason?: unknown
      }>(
        `${BASE_URL}/${params.agentNodeId}/logEntries`,
        { params: queryParams, signal: params.signal },
      )
      payload = response.data ?? {}
    } catch (error) {
      if (isCanceledRequestError(error)) {
        throw error
      }
      throw toLogQueryError(error)
    }

    const logs = Array.isArray(payload.results)
      ? payload.results.map((item) => normalizeLogItem(item)).filter((item): item is AgentLogItem => item !== null)
      : []
    const nextCursor = asString(payload.nextPageToken)
    const previousCursor = asString(payload.previousPageToken)

    return {
      logs,
      nextCursor,
      previousCursor,
      hasOlder: asBoolean(payload.hasOlder),
      hasNewer: asBoolean(payload.hasNewer),
      // Backend owns viewer transport truth. Missing metadata must stay conservative.
      caughtUp: payload.caughtUp === true,
      gap: asBoolean(payload.gap),
      gapReason: asString(payload.gapReason),
    }
  },
}

function parseAgentRegistrationSafe(resourceName: string): number {
  try {
    return parseAgentName(resourceName)
  } catch {
    return invalidAgentResponse('agent.name')
  }
}

function parseRegistrationTokenSafe(resourceName: string): number {
  try {
    return parseAgentRegistrationTokenName(resourceName)
  } catch {
    return invalidAgentResponse('registrationToken.name')
  }
}

function isCanceledRequestError(error: unknown): boolean {
  if (error instanceof DOMException) {
    return error.name === "AbortError"
  }
  if (!(error instanceof Error)) {
    return false
  }
  const maybeCanceled = error as Error & { code?: unknown }
  return error.name === "AbortError" || error.name === "CanceledError" || maybeCanceled.code === "ERR_CANCELED"
}

function toLogQueryError(error: unknown): AgentLogQueryError {
  const axiosError = error as AxiosError<{
    error?: {
      code?: string
      message?: string
    }
  }>

  const status = axiosError?.response?.status
  let code = typeof status === 'number' ? `http_${status}` : 'network_error'
  let message = axiosError?.message || 'Request failed'

  const body = axiosError?.response?.data
  if (body?.error?.code) {
    code = body.error.code
  }
  if (body?.error?.message) {
    message = body.error.message
  }

  return new AgentLogQueryError(code, message, status)
}

function normalizeLogItem(raw: unknown): AgentLogItem | null {
  if (!raw || typeof raw !== 'object') {
    return null
  }
  const item = raw as Partial<AgentLogItem>
  const id = asString(item.id)
  if (!id) {
    return null
  }
  return {
    id,
    ts: asString(item.ts),
    tsNs: asString(item.tsNs),
    stream: asString(item.stream) || 'stdout',
    line: asString(item.line),
    truncated: Boolean(item.truncated),
  }
}

function asString(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

function asBoolean(value: unknown): boolean {
  return value === true
}

function requireLogLimit(value: number | undefined): number {
  const limit = value ?? DEFAULT_LOG_LIMIT
  if (!Number.isInteger(limit) || limit < 1 || limit > MAX_LOG_LIMIT) {
    throw new AgentLogQueryError("bad_request", `pageSize must be between 1 and ${MAX_LOG_LIMIT}`)
  }
  return limit
}
