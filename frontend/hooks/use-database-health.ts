import { useQuery } from '@tanstack/react-query'
import { getDatabaseHealth } from '@/services/database-health.service'

export function useDatabaseHealth() {
  return useQuery({
    queryKey: ['system', 'database-health'],
    queryFn: () => getDatabaseHealth(),
    refetchInterval: 10000,
  })
}
