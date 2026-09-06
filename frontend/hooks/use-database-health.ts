import { useQuery } from '@tanstack/react-query'
import { getDatabaseHealth } from '@/services/database-health.service'

export const databaseHealthKeys = {
  current: () => ['system', 'database-health'] as const,
}

export function useDatabaseHealth() {
  return useQuery({
    queryKey: databaseHealthKeys.current(),
    queryFn: () => getDatabaseHealth(),
    refetchInterval: 60000,
  })
}
