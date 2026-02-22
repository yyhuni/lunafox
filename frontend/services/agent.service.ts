/**
 * Agent management API service (Go backend)
 */

import { api } from '@/lib/api-client'
import type {
  Agent,
  AgentsResponse,
  RegistrationTokenResponse,
  UpdateAgentConfigRequest,
} from '@/types/agent.types'
import { USE_MOCK, mockDelay, getMockAgents, getMockAgentById, getMockRegistrationToken } from '@/mock'

const BASE_URL = '/admin/agents'

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
}
