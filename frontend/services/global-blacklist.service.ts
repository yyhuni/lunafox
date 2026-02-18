import { api } from '@/lib/api-client'
import { USE_MOCK, mockDelay, getMockGlobalBlacklist, updateMockGlobalBlacklist } from '@/mock'

export interface GlobalBlacklistResponse {
  patterns: string[]
}

export interface UpdateGlobalBlacklistRequest {
  patterns: string[]
}

/**
 * Get global blacklist rules
 */
export async function getGlobalBlacklist(): Promise<GlobalBlacklistResponse> {
  if (USE_MOCK) {
    await mockDelay()
    return getMockGlobalBlacklist()
  }
  const res = await api.get<GlobalBlacklistResponse>('/blacklist/rules/')
  return res.data
}

/**
 * Update global blacklist rules (full replace)
 */
export async function updateGlobalBlacklist(data: UpdateGlobalBlacklistRequest): Promise<GlobalBlacklistResponse> {
  if (USE_MOCK) {
    await mockDelay()
    return updateMockGlobalBlacklist(data)
  }
  const res = await api.put<GlobalBlacklistResponse>('/blacklist/rules/', data)
  return res.data
}
