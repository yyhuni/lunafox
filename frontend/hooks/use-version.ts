import { useCallback } from "react"
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { VersionService } from '@/services/version.service'

interface UseVersionOptions {
  enabled?: boolean
}

export function useVersion({ enabled = true }: UseVersionOptions = {}) {
  return useQuery({
    queryKey: ['version'],
    queryFn: () => VersionService.getVersion(),
    enabled,
    staleTime: Infinity,
  })
}

export function useCheckUpdate() {
  return useQuery({
    queryKey: ['check-update'],
    queryFn: () => VersionService.checkUpdate(),
    enabled: false, // manual trigger
    staleTime: 5 * 60 * 1000, // 5 minutes cache
  })
}

export function useCheckUpdateAction() {
  const queryClient = useQueryClient()

  return useCallback(() => {
    return queryClient.fetchQuery({
      queryKey: ['check-update'],
      queryFn: () => VersionService.checkUpdate(),
      staleTime: 5 * 60 * 1000,
    })
  }, [queryClient])
}
