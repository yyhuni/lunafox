/**
 * Agent management API service (Go backend)
 */

import type { AxiosError } from 'axios'
import { api } from '@/lib/api-client'
import type {
  Agent,
  AgentsResponse,
  RegistrationTokenResponse,
  UpdateAgentConfigRequest,
} from '@/types/agent.types'
import type { AgentLogItem, AgentLogsResponse } from '@/types/agent-log.types'
import { USE_MOCK, mockDelay, getMockAgents, getMockAgentById, getMockRegistrationToken } from '@/mock'

const BASE_URL = '/admin/agents'

const DEFAULT_LOG_LIMIT = 200
const MAX_LOG_LIMIT = 500

export interface FetchAgentLogsParams {
  agentId: number
  container: string
  limit?: number
  cursor?: string
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

export const agentService = {
  async getAgents(page = 1, pageSize = 10, status?: string): Promise<AgentsResponse> {
    if (USE_MOCK) {
      await mockDelay()
      return getMockAgents(page, pageSize, status)
    }
    const params: Record<string, number | string> = { page, pageSize, include: 'heartbeat' }
    if (status) params.status = status
    const response = await api.get<AgentsResponse>(BASE_URL, { params })
    return response.data
  },

  async getAgent(id: number): Promise<Agent> {
    if (USE_MOCK) {
      await mockDelay()
      const agent = getMockAgentById(id)
      if (!agent) throw new Error('Agent not found')
      return agent
    }
    const response = await api.get<Agent>(`${BASE_URL}/${id}`)
    return response.data
  },

  async deleteAgent(id: number): Promise<void> {
    await api.delete(`${BASE_URL}/${id}`)
  },

  async updateAgentConfig(id: number, data: UpdateAgentConfigRequest): Promise<Agent> {
    const response = await api.put<Agent>(`${BASE_URL}/${id}/config`, data)
    return response.data
  },

  async createRegistrationToken(): Promise<RegistrationTokenResponse> {
    if (USE_MOCK) {
      await mockDelay()
      return getMockRegistrationToken()
    }
    const response = await api.post<RegistrationTokenResponse>(`${BASE_URL}/registration-tokens`)
    return response.data
  },

  async fetchAgentLogs(params: FetchAgentLogsParams): Promise<AgentLogsResponse> {
    if (USE_MOCK) {
      await mockDelay(120)
      const ts = new Date().toISOString()
      const tsNs = `${Date.now()}000000`
      return {
        logs: [
          {
            id: `mock:${params.agentId}:${tsNs}`,
            ts,
            tsNs,
            stream: 'stdout',
            line: '[mock] log polling is enabled in mock mode',
            truncated: false,
          },
        ],
        nextCursor: params.cursor?.trim() || `mock-cursor:${params.agentId}:${tsNs}`,
        hasMore: false,
      }
    }

    const container = params.container.trim()
    if (!container) {
      throw new AgentLogQueryError('bad_request', 'Container is required')
    }

    const limit = Math.min(Math.max(params.limit ?? DEFAULT_LOG_LIMIT, 1), MAX_LOG_LIMIT)
    const cursor = params.cursor?.trim() ?? ''
    const queryParams: Record<string, string> = {
      container,
      limit: String(limit),
    }
    if (cursor) {
      queryParams.cursor = cursor
    }

    let payload: Partial<AgentLogsResponse>
    try {
      const response = await api.get<Partial<AgentLogsResponse>>(`${BASE_URL}/${params.agentId}/logs`, {
        params: queryParams,
        signal: params.signal,
      })
      payload = response.data ?? {}
    } catch (error) {
      throw toLogQueryError(error)
    }

    return {
      logs: Array.isArray(payload.logs)
        ? payload.logs.map((item) => normalizeLogItem(item)).filter((item): item is AgentLogItem => item !== null)
      : [],
      nextCursor: asString(payload.nextCursor),
      hasMore: Boolean(payload.hasMore),
    }
  },
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
