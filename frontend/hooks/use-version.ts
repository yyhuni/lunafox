import { useCallback, useEffect, useMemo, useState } from "react"
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { VersionService, isUpgradeNetworkError, toUpgradeApiError } from "@/services/version.service"
import type {
  CreateUpgradeOperationInput,
  UpgradeOperation,
  UpgradeOperationStatus,
  UpgradeUserStage,
  UpdateCheckResult,
  VersionInfo,
} from "@/types/version.types"

export const UPGRADE_OPERATION_STORAGE_KEY = "lunafox.upgrade.operationId"
export const UPGRADE_PENDING_SUCCESS_STORAGE_KEY = "lunafox.upgrade.pendingSuccessOperationId"
export const UPGRADE_OPERATION_CHANGED_EVENT = "lunafox:upgrade-operation-changed"
const UPGRADE_COMPLETION_ACK_PREFIX = "lunafox.upgrade.completionAcknowledged."
const ACTIVE_POLL_BASE_MS = 1_500
const ACTIVE_POLL_MAX_MS = 15_000

interface UseVersionOptions {
	enabled?: boolean
}

interface UseCheckForUpdatesOptions {
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

export function useCheckForUpdates({ enabled = false }: UseCheckForUpdatesOptions = {}) {
  return useQuery<UpdateCheckResult>({
    queryKey: ["upgrade", "checkForUpdates"],
    queryFn: VersionService.checkForUpdates,
    enabled,
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
    window.dispatchEvent(new Event(UPGRADE_OPERATION_CHANGED_EVENT))
  } catch {
    // The server operation remains durable when browser storage is unavailable.
  }
}

export function clearStoredUpgradeOperationId(operationId?: string): void {
  if (typeof window === "undefined") return
  try {
    if (!operationId || window.localStorage.getItem(UPGRADE_OPERATION_STORAGE_KEY) === operationId) {
      window.localStorage.removeItem(UPGRADE_OPERATION_STORAGE_KEY)
      window.dispatchEvent(new Event(UPGRADE_OPERATION_CHANGED_EVENT))
    }
  } catch {
    // The server operation remains durable when browser storage is unavailable.
  }
}

export function persistPendingSuccessOperationId(operationId: string): void {
  if (typeof window === "undefined") return
  try {
    window.localStorage.setItem(UPGRADE_PENDING_SUCCESS_STORAGE_KEY, operationId)
  } catch {
    // The success notice is best-effort; the durable Operation remains authoritative.
  }
}

export function readPendingSuccessOperationId(): string | null {
  if (typeof window === "undefined") return null
  try {
    return window.localStorage.getItem(UPGRADE_PENDING_SUCCESS_STORAGE_KEY)
  } catch {
    return null
  }
}

export function clearPendingSuccessOperationId(operationId?: string): void {
  if (typeof window === "undefined") return
  try {
    if (!operationId || window.localStorage.getItem(UPGRADE_PENDING_SUCCESS_STORAGE_KEY) === operationId) {
      window.localStorage.removeItem(UPGRADE_PENDING_SUCCESS_STORAGE_KEY)
    }
  } catch {
    // Storage failures must not change the server-backed upgrade result.
  }
}

export function hasShownUpgradeCompletion(operationId: string): boolean {
  if (typeof window === "undefined") return false
  try {
    return window.localStorage.getItem(`${UPGRADE_COMPLETION_ACK_PREFIX}${operationId}`) === "1"
  } catch {
    return false
  }
}

export function markUpgradeCompletionShown(operationId: string): void {
  if (typeof window === "undefined") return
  try {
    window.localStorage.setItem(`${UPGRADE_COMPLETION_ACK_PREFIX}${operationId}`, "1")
  } catch {
    // Completion acknowledgement is a UX convenience, not the operation source of truth.
  }
}

export function isUpgradeOperationTerminal(status: UpgradeOperation["status"] | undefined): boolean {
  return status === "succeeded" || status === "failed" || status === "needs_recovery" || status === "needs_attention"
}

export function isUpgradeOperationActive(status: UpgradeOperation["status"] | undefined): boolean {
  return Boolean(status) && !isUpgradeOperationTerminal(status)
}

export function upgradeUserStageForStatus(status: UpgradeOperationStatus | undefined): UpgradeUserStage {
  switch (status) {
    case "stopping":
      return "stopping"
    case "updating":
    case "migrating":
      return "updating"
    case "restarting":
      return "restarting"
    case "agent_verifying":
    case "verifying":
      return "verifying"
    case "succeeded":
    case "failed":
    case "needs_recovery":
    case "needs_attention":
      return "finished"
    case "queued":
    case "preflight":
    default:
      return "preparing"
  }
}

export function upgradeOperationPollDelay(failureCount: number): number {
  const exponent = Math.max(0, Math.min(4, Number.isFinite(failureCount) ? failureCount : 0))
  return Math.min(ACTIVE_POLL_MAX_MS, ACTIVE_POLL_BASE_MS * (2 ** exponent))
}

export function useUpgradeOperation(operationId?: string | null, options: UseUpgradeOperationOptions = {}) {
	const { enabled = true } = options
	const autoResolve = operationId === undefined
	const [storedOperationId, setStoredOperationId] = useState<string | null>(readStoredUpgradeOperationId)
  const requestedOperationId = autoResolve ? storedOperationId : operationId
  const queryKeyId = requestedOperationId ?? (autoResolve ? "active" : "none")

  useEffect(() => {
    const onStorage = (event: StorageEvent) => {
      if (event.key === UPGRADE_OPERATION_STORAGE_KEY) setStoredOperationId(event.newValue)
    }
    const onOperationChanged = () => setStoredOperationId(readStoredUpgradeOperationId())
    window.addEventListener("storage", onStorage)
    window.addEventListener(UPGRADE_OPERATION_CHANGED_EVENT, onOperationChanged)
    return () => {
      window.removeEventListener("storage", onStorage)
      window.removeEventListener(UPGRADE_OPERATION_CHANGED_EVENT, onOperationChanged)
    }
  }, [])

  const query = useQuery<UpgradeOperation | null>({
    queryKey: ["upgrade", "operation", queryKeyId],
    queryFn: async () => {
      if (autoResolve) {
        // The active view is the route-lock authority. A browser id is only a
        // fallback for showing a terminal result after the active view is empty.
        const active = await VersionService.getActiveUpgradeOperation()
        if (active) return active
        if (requestedOperationId) {
          try {
            return await VersionService.getUpgradeOperation(requestedOperationId)
          } catch (error) {
            const parsed = toUpgradeApiError(error)
            if (parsed.status !== 404) throw parsed
          }
        }
        return null
      }
      if (requestedOperationId) {
        try {
          return await VersionService.getUpgradeOperation(requestedOperationId)
        } catch (error) {
          throw toUpgradeApiError(error)
        }
      }
      return null
    },
    enabled: enabled && (autoResolve || Boolean(requestedOperationId)),
    staleTime: 0,
    refetchInterval: (current) => {
      // A successful empty active view is a stable "nothing to do" result;
      // polling it forever creates needless traffic and can keep a route in
      // an ambiguous resolving state.
      if (current.state.data === null) return false
      if (isUpgradeOperationTerminal(current.state.data?.status)) return false
      return upgradeOperationPollDelay(current.state.fetchFailureCount)
    },
    refetchIntervalInBackground: true,
    retry: (failureCount, error) => isUpgradeNetworkError(error) && failureCount < 3,
  })

  useEffect(() => {
    const discoveredId = query.data?.operationId
    if (discoveredId) {
      if (autoResolve && discoveredId !== storedOperationId) {
        setStoredOperationId(discoveredId)
        persistUpgradeOperationId(discoveredId)
      }
      if (query.data?.status === "succeeded") persistPendingSuccessOperationId(discoveredId)
      return
    }
    if (autoResolve && query.isSuccess && query.data === null && storedOperationId) {
      clearStoredUpgradeOperationId(storedOperationId)
      setStoredOperationId(null)
    }
  }, [autoResolve, query.data, query.isSuccess, storedOperationId])

  return useMemo(() => ({
    ...query,
    data: query.data ?? undefined,
    operationId: query.data?.operationId ?? requestedOperationId ?? null,
    isReconnecting: Boolean((requestedOperationId || autoResolve) && query.isError && isUpgradeNetworkError(query.error)),
    isResolving: Boolean(enabled && (autoResolve || requestedOperationId) && query.isPending),
    isActive: isUpgradeOperationActive(query.data?.status),
    userStage: upgradeUserStageForStatus(query.data?.status),
    lastConfirmedStage: query.data?.status,
  }), [autoResolve, enabled, query, requestedOperationId])
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
      queryClient.setQueryData(["upgrade", "operation", "active"], operation)
    },
    onError: (_error, operationId) => {
      void queryClient.invalidateQueries({ queryKey: ["upgrade", "operation", operationId] })
      void queryClient.invalidateQueries({ queryKey: ["upgrade", "operation", "active"] })
    },
  })
}

export function useStopUpgradeOperation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (operationId: string) => VersionService.stopUpgradeOperation(operationId),
    onSuccess: (operation) => {
      persistUpgradeOperationId(operation.operationId)
      queryClient.setQueryData(["upgrade", "operation", operation.operationId], operation)
      queryClient.setQueryData(["upgrade", "operation", "active"], operation)
    },
    onError: (_error, operationId) => {
      // A host may have accepted cancellation before the response was lost;
      // refetch both views so the status page cannot keep showing stale work.
      void queryClient.invalidateQueries({ queryKey: ["upgrade", "operation", operationId] })
      void queryClient.invalidateQueries({ queryKey: ["upgrade", "operation", "active"] })
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
