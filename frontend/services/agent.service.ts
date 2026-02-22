/**
 * Agent management API service (Go backend)
 */

import { api, tokenManager } from '@/lib/api-client'
import { getBackendBaseUrl } from '@/lib/env'
import type {
  Agent,
  AgentsResponse,
  RegistrationTokenResponse,
  UpdateAgentConfigRequest,
} from '@/types/agent.types'
import type {
  AgentLogChunkEvent,
  AgentLogDoneEvent,
  AgentLogErrorEvent,
  AgentLogPingEvent,
  AgentLogStatusEvent,
  AgentLogStreamEvent,
} from '@/types/agent-log.types'
import { USE_MOCK, mockDelay, getMockAgents, getMockAgentById, getMockRegistrationToken } from '@/mock'

const BASE_URL = '/admin/agents'

const MAX_TAIL = 2000

export interface StreamAgentLogsParams {
  agentId: number
  container: string
  tail?: number
  follow?: boolean
  timestamps?: boolean
  signal?: AbortSignal
  onEvent: (event: AgentLogStreamEvent) => void
}

export class AgentLogStreamError extends Error {
  code: string
  status?: number

  constructor(code: string, message: string, status?: number) {
    super(message)
    this.name = 'AgentLogStreamError'
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

  async streamAgentLogs(params: StreamAgentLogsParams): Promise<void> {
    if (USE_MOCK) {
      await mockDelay(300)
      params.onEvent({ type: 'status', requestId: 'mock-1', status: 'started' })
      params.onEvent({
        type: 'log',
        requestId: 'mock-1',
        ts: new Date().toISOString(),
        stream: 'stdout',
        line: '[mock] log stream is enabled in mock mode',
        truncated: false,
      })
      params.onEvent({ type: 'done', requestId: 'mock-1', reason: 'eof' })
      return
    }

    const container = params.container.trim()
    if (!container) {
      throw new AgentLogStreamError('BAD_REQUEST', 'Container is required')
    }

    const tail = Math.min(Math.max(params.tail ?? 200, 0), MAX_TAIL)
    const follow = params.follow ?? true
    const timestamps = params.timestamps ?? true

    const query = new URLSearchParams({
      container,
      tail: String(tail),
      follow: String(follow),
      timestamps: String(timestamps),
    })

    const token = tokenManager.getAccessToken()
    const headers: HeadersInit = {
      Accept: 'text/event-stream',
    }
    if (token) {
      headers.Authorization = `Bearer ${token}`
    }

    const response = await fetch(
      `${getBackendBaseUrl()}/api/admin/agents/${params.agentId}/logs/stream?${query.toString()}`,
      {
        method: 'GET',
        headers,
        signal: params.signal,
        cache: 'no-store',
      }
    )

    if (!response.ok) {
      throw await toStreamError(response)
    }

    const reader = response.body?.getReader()
    if (!reader) {
      throw new AgentLogStreamError('STREAM_UNAVAILABLE', 'ReadableStream is not available')
    }

    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) {
        if (buffer.trim()) {
          const event = parseSSEFrame(buffer)
          if (event) {
            params.onEvent(event)
          }
        }
        return
      }

      buffer += decoder.decode(value, { stream: true })
      buffer = buffer.replace(/\r\n/g, '\n')

      let boundary = buffer.indexOf('\n\n')
      while (boundary !== -1) {
        const frame = buffer.slice(0, boundary)
        buffer = buffer.slice(boundary + 2)

        const event = parseSSEFrame(frame)
        if (event) {
          params.onEvent(event)
          if (event.type === 'done') {
            return
          }
        }

        boundary = buffer.indexOf('\n\n')
      }
    }
  },
}

async function toStreamError(response: Response): Promise<AgentLogStreamError> {
  let code = `HTTP_${response.status}`
  let message = `Request failed with status ${response.status}`

  try {
    const data = (await response.json()) as {
      error?: {
        code?: string
        message?: string
      }
    }
    if (data?.error?.code) {
      code = data.error.code
    }
    if (data?.error?.message) {
      message = data.error.message
    }
  } catch {
    // ignore non-json error body
  }

  return new AgentLogStreamError(code, message, response.status)
}

function parseSSEFrame(frame: string): AgentLogStreamEvent | null {
  if (!frame.trim()) {
    return null
  }

  let eventType = 'message'
  const dataLines: string[] = []

  for (const line of frame.split('\n')) {
    if (line.startsWith('event:')) {
      eventType = line.slice(6).trim()
      continue
    }
    if (line.startsWith('data:')) {
      dataLines.push(line.slice(5).trim())
    }
  }

  if (!dataLines.length) {
    return null
  }

  const rawData = dataLines.join('\n')
  try {
    const data = JSON.parse(rawData) as Record<string, unknown>
    return mapSSEEvent(eventType, data)
  } catch {
    return null
  }
}

function mapSSEEvent(type: string, data: Record<string, unknown>): AgentLogStreamEvent | null {
  const requestId = asString(data.requestId)
  if (!requestId) {
    return null
  }

  switch (type) {
    case 'status': {
      const event: AgentLogStatusEvent = {
        type: 'status',
        requestId,
        status: asString(data.status) || 'unknown',
      }
      return event
    }
    case 'log': {
      const event: AgentLogChunkEvent = {
        type: 'log',
        requestId,
        ts: asString(data.ts) || new Date().toISOString(),
        stream: asString(data.stream) || 'stdout',
        line: asString(data.line) || '',
        truncated: Boolean(data.truncated),
      }
      return event
    }
    case 'error': {
      const event: AgentLogErrorEvent = {
        type: 'error',
        requestId,
        code: asString(data.code) || 'internal_error',
        message: asString(data.message) || 'stream error',
      }
      return event
    }
    case 'done': {
      const event: AgentLogDoneEvent = {
        type: 'done',
        requestId,
        reason: asString(data.reason) || 'done',
      }
      return event
    }
    case 'ping': {
      const event: AgentLogPingEvent = {
        type: 'ping',
        requestId,
        ts: asString(data.ts) || new Date().toISOString(),
      }
      return event
    }
    default:
      return null
  }
}

function asString(value: unknown): string {
  return typeof value === 'string' ? value : ''
}
