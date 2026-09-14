import { useCallback, useEffect, useMemo, useState } from "react"
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { VersionService, isUpgradeNetworkError, toUpgradeApiError } from "@/services/version.service"
import type { CreateUpgradeOperationInput, UpgradeOperation, UpdateCheckResult, VersionInfo } from "@/types/version.types"

export const UPGRADE_OPERATION_STORAGE_KEY = "lunafox.upgrade.operationId"
const ACTIVE_POLL_BASE_MS = 1_500
const ACTIVE_POLL_MAX_MS = 15_000

interface UseVersionOptions {
	enabled?: boolean
}

interface UseUpgradeOperationOptions {
	enabled?: boolean
}

export function useVersion({ enabled = true }: UseVersionOptions = {}) {
  return useQuery<VersionInfo>({
    queryKey: ["version"],
    queryFn: async () => {
      const result = await VersionService.checkForUpdates()
      return { version: result.currentVersion, githubRepo: "https://github.com/yyhuni/lunafox" }
    },
    enabled,
    staleTime: 5 * 60 * 1000,
  })
}

export function useCheckForUpdates() {
  return useQuery<UpdateCheckResult>({
    queryKey: ["upgrade", "checkForUpdates"],
    queryFn: VersionService.checkForUpdates,
    enabled: false,
    staleTime: 0,
  })
}

export function useCheckForUpdatesAction() {
  const queryClient = useQueryClient()
  return useCallback(() => queryClient.fetchQuery({
    queryKey: ["upgrade", "checkForUpdates"],
    queryFn: VersionService.checkForUpdates,
    staleTime: 0,
  }), [queryClient])
}

export function readStoredUpgradeOperationId(): string | null {
  if (typeof window === "undefined") return null
  try {
    return window.localStorage.getItem(UPGRADE_OPERATION_STORAGE_KEY)
  } catch {
    return null
  }
}

export function persistUpgradeOperationId(operationId: string): void {
  if (typeof window === "undefined") return
  try {
    window.localStorage.setItem(UPGRADE_OPERATION_STORAGE_KEY, operationId)
  } catch {
    // The server operation remains durable when browser storage is unavailable.
  }
}

export function isUpgradeOperationTerminal(status: UpgradeOperation["status"] | undefined): boolean {
  return status === "succeeded" || status === "failed" || status === "needs_recovery" || status === "needs_attention"
}

export function upgradeOperationPollDelay(failureCount: number): number {
  const exponent = Math.max(0, Math.min(4, Number.isFinite(failureCount) ? failureCount : 0))
  return Math.min(ACTIVE_POLL_MAX_MS, ACTIVE_POLL_BASE_MS * (2 ** exponent))
}

export function useUpgradeOperation(operationId?: string | null, options: UseUpgradeOperationOptions = {}) {
	const { enabled = true } = options
	const [storedOperationId, setStoredOperationId] = useState<string | null>(readStoredUpgradeOperationId)
  const resolvedOperationId = operationId === undefined ? storedOperationId : operationId

  useEffect(() => {
    const onStorage = (event: StorageEvent) => {
      if (event.key === UPGRADE_OPERATION_STORAGE_KEY) setStoredOperationId(event.newValue)
    }
    window.addEventListener("storage", onStorage)
    return () => window.removeEventListener("storage", onStorage)
  }, [])

  const query = useQuery<UpgradeOperation>({
    queryKey: ["upgrade", "operation", resolvedOperationId],
    queryFn: () => VersionService.getUpgradeOperation(resolvedOperationId as string),
    enabled: enabled && Boolean(resolvedOperationId),
    staleTime: 0,
    refetchInterval: (current) => {
      if (isUpgradeOperationTerminal(current.state.data?.status)) return false
      return upgradeOperationPollDelay(current.state.fetchFailureCount)
    },
    refetchIntervalInBackground: true,
    retry: (failureCount, error) => isUpgradeNetworkError(error) && failureCount < 3,
  })

  return useMemo(() => ({
    ...query,
    operationId: resolvedOperationId,
    isReconnecting: Boolean(resolvedOperationId && query.isError && isUpgradeNetworkError(query.error)),
    lastConfirmedStage: query.data?.status,
  }), [query, resolvedOperationId])
}

export function useCreateUpgradeOperation() {
  const queryClient = useQueryClient()
  const [operationId, setOperationId] = useState<string | null>(readStoredUpgradeOperationId)
  const mutation = useMutation({
    mutationFn: (input: CreateUpgradeOperationInput) => VersionService.createUpgradeOperation(input),
    onSuccess: (operation) => {
      persistUpgradeOperationId(operation.operationId)
      setOperationId(operation.operationId)
      queryClient.setQueryData(["upgrade", "operation", operation.operationId], operation)
    },
  })
  return { ...mutation, operationId, setOperationId }
}

export function useRetryUpgradeOperation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (operationId: string) => VersionService.retryUpgradeOperation(operationId),
    onSuccess: (operation) => {
      persistUpgradeOperationId(operation.operationId)
      queryClient.setQueryData(["upgrade", "operation", operation.operationId], operation)
    },
  })
}

export function getUpgradeErrorMessage(error: unknown): string {
  return toUpgradeApiError(error).message
}

// Callers using the former names still use the canonical action and resource
// routes; no legacy endpoint fallback is retained.
export const useCheckUpdate = useCheckForUpdates
export const useCheckUpdateAction = useCheckForUpdatesAction
