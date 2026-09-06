import { useQuery } from "@tanstack/react-query"

import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import { getErrorCode, getErrorResponseData } from "@/lib/response-parser"
import {
  getGlobalBlacklistPolicy,
  getTargetBlacklistPolicy,
  updateGlobalBlacklistPolicy,
  updateTargetBlacklistPolicy,
} from "@/services/blacklist-policy.service"
import type { UpdateBlacklistPolicyInput } from "@/types/blacklist-policy.types"

export const blacklistPolicyKeys = {
  all: ["blacklistPolicy"] as const,
  global: () => [...blacklistPolicyKeys.all, "global"] as const,
  target: (targetId: number) => [...blacklistPolicyKeys.all, "target", targetId] as const,
}

function isBlacklistPolicyConflict(error: unknown): boolean {
  const status = typeof error === "object" && error !== null && "response" in error
    ? (error as { response?: { status?: unknown } }).response?.status
    : undefined
  return status === 409 || getErrorCode(getErrorResponseData(error)) === "ABORTED"
}

async function handlePolicyMutationError({
  error,
  queryClient,
  toast,
  queryKey,
}: {
  error: unknown
  queryClient: {
    invalidateQueries: (filters: {
      queryKey: readonly unknown[]
      exact: boolean
      refetchType: "active"
    }) => Promise<unknown>
  }
  toast: {
    errorFromCode: (code: string | null, fallbackKey?: string) => void
    warning: (key: string) => void
  }
  queryKey: readonly unknown[]
}) {
  if (isBlacklistPolicyConflict(error)) {
    toast.warning("pages.settings.blacklist.toast.conflict")
    await queryClient.invalidateQueries({ queryKey, exact: true, refetchType: "active" })
    return
  }
  toast.errorFromCode(getErrorCode(getErrorResponseData(error)), "pages.settings.blacklist.toast.saveError")
}

export function useGlobalBlacklistPolicy() {
  return useQuery({
    queryKey: blacklistPolicyKeys.global(),
    queryFn: getGlobalBlacklistPolicy,
  })
}

export function useTargetBlacklistPolicy(targetId: number) {
  return useQuery({
    queryKey: blacklistPolicyKeys.target(targetId),
    queryFn: () => getTargetBlacklistPolicy(targetId),
    enabled: !!targetId,
  })
}

export function useUpdateGlobalBlacklistPolicy() {
  const queryKey = blacklistPolicyKeys.global()
  return useResourceMutation({
    mutationFn: (input: UpdateBlacklistPolicyInput) => updateGlobalBlacklistPolicy(input),
    // A stale etag must terminate this edit. Do not let a transport default
    // repeat the replacement after the Server has rejected its freshness.
    retry: false,
    loadingToast: {
      key: "common.status.updating",
      params: {},
      id: "update-global-blacklist-policy",
    },
    invalidate: [{ queryKey, exact: true }],
    onSuccess: ({ toast }) => {
      toast.success("pages.settings.blacklist.toast.saveSuccess")
    },
    onError: ({ error, queryClient, toast }) => handlePolicyMutationError({
      error,
      queryClient,
      toast,
      queryKey,
    }),
  })
}

export function useUpdateTargetBlacklistPolicy() {
  return useResourceMutation({
    mutationFn: ({ targetId, ...input }: { targetId: number } & UpdateBlacklistPolicyInput) => (
      updateTargetBlacklistPolicy(targetId, input)
    ),
    // Target-local replacement has the same etag conflict boundary as global policy.
    retry: false,
    loadingToast: {
      key: "common.status.updating",
      params: {},
      id: ({ targetId }) => `update-target-blacklist-policy-${targetId}`,
    },
    invalidate: [
      ({ variables }) => ({ queryKey: blacklistPolicyKeys.target(variables.targetId), exact: true }),
    ],
    onSuccess: ({ toast }) => {
      toast.success("pages.settings.blacklist.toast.saveSuccess")
    },
    onError: ({ error, variables, queryClient, toast }) => handlePolicyMutationError({
      error,
      queryClient,
      toast,
      queryKey: blacklistPolicyKeys.target(variables.targetId),
    }),
  })
}
