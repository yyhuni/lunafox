import { api } from '@/lib/api-client'
import type { DatabaseHealthSnapshot } from '@/types/database-health.types'

export async function getDatabaseHealth(): Promise<DatabaseHealthSnapshot> {
  const res = await api.get<DatabaseHealthSnapshot>('/databaseHealthReports/current')
  return res.data
}
