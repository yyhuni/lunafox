import { api } from '@/lib/api-client'
import type { DatabaseHealthSnapshot } from '@/types/database-health.types'
import { USE_MOCK, mockDelay, mockDatabaseHealth } from '@/mock'

export async function getDatabaseHealth(): Promise<DatabaseHealthSnapshot> {
  if (USE_MOCK) {
    await mockDelay()
    return mockDatabaseHealth
  }
  const res = await api.get<DatabaseHealthSnapshot>('/system/database-health/')
  return res.data
}
