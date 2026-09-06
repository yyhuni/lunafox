import { act, waitFor } from "@testing-library/react"
import { QueryClient } from "@tanstack/react-query"
import { beforeEach, describe, expect, it, vi } from "vitest"

const blacklistPolicyServiceMocks = vi.hoisted(() => ({
  getGlobalBlacklistPolicy: vi.fn(),
  getTargetBlacklistPolicy: vi.fn(),
  updateGlobalBlacklistPolicy: vi.fn(),
  updateTargetBlacklistPolicy: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/blacklist-policy.service", () => blacklistPolicyServiceMocks)

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

import {
  blacklistPolicyKeys,
  useGlobalBlacklistPolicy,
  useTargetBlacklistPolicy,
  useUpdateGlobalBlacklistPolicy,
  useUpdateTargetBlacklistPolicy,
} from "@/hooks/use-blacklist-policy"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const globalPolicy = {
  name: "blacklistPolicy",
  patterns: ["*.example.com"],
  etag: "global-etag",
  updateTime: "2026-08-05T00:00:00Z",
}

const targetPolicy = {
  name: "targets/9/blacklistPolicy",
  patterns: ["192.0.2.0/24"],
  etag: "target-etag",
  updateTime: "2026-08-05T00:00:00Z",
}

describe("use-blacklist-policy", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("keeps global and Target-local policy queries in distinct caches", async () => {
    blacklistPolicyServiceMocks.getGlobalBlacklistPolicy.mockResolvedValue(globalPolicy)
    blacklistPolicyServiceMocks.getTargetBlacklistPolicy.mockResolvedValue(targetPolicy)

    const queryClient = createTestQueryClient()
    const { result } = renderHookWithProviders(() => ({
      global: useGlobalBlacklistPolicy(),
      target: useTargetBlacklistPolicy(9),
    }), { queryClient })

    await waitFor(() => {
      expect(result.current.global.data).toEqual(globalPolicy)
      expect(result.current.target.data).toEqual(targetPolicy)
    })

    expect(blacklistPolicyServiceMocks.getGlobalBlacklistPolicy).toHaveBeenCalledTimes(1)
    expect(blacklistPolicyServiceMocks.getTargetBlacklistPolicy).toHaveBeenCalledWith(9)
    expect(queryClient.getQueryData(blacklistPolicyKeys.global())).toEqual(globalPolicy)
    expect(queryClient.getQueryData(blacklistPolicyKeys.target(9))).toEqual(targetPolicy)
    expect(blacklistPolicyKeys.global()).not.toEqual(blacklistPolicyKeys.target(9))
  })

  it("saves global policy with one pending/success lifecycle and an exact cache invalidation", async () => {
    blacklistPolicyServiceMocks.updateGlobalBlacklistPolicy.mockResolvedValue(globalPolicy)

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateGlobalBlacklistPolicy(), { queryClient })

    await act(async () => {
      await result.current.mutateAsync({ patterns: ["*.example.com"], etag: "global-etag" })
    })

    expect(blacklistPolicyServiceMocks.updateGlobalBlacklistPolicy).toHaveBeenCalledWith({
      patterns: ["*.example.com"],
      etag: "global-etag",
    })
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.updating",
      {},
      "update-global-blacklist-policy"
    )
    expect(toastMocks.success).toHaveBeenCalledWith(
      "pages.settings.blacklist.toast.saveSuccess",
      undefined,
      "update-global-blacklist-policy"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: blacklistPolicyKeys.global(),
      exact: true,
    })
  })

  it("saves only the Target-local policy and invalidates only its child key", async () => {
    blacklistPolicyServiceMocks.updateTargetBlacklistPolicy.mockResolvedValue(targetPolicy)

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateTargetBlacklistPolicy(), { queryClient })

    await act(async () => {
      await result.current.mutateAsync({
        targetId: 9,
        patterns: ["192.0.2.0/24"],
        etag: "target-etag",
      })
    })

    expect(blacklistPolicyServiceMocks.updateTargetBlacklistPolicy).toHaveBeenCalledWith(9, {
      patterns: ["192.0.2.0/24"],
      etag: "target-etag",
    })
    expect(toastMocks.success).toHaveBeenCalledWith(
      "pages.settings.blacklist.toast.saveSuccess",
      undefined,
      "update-target-blacklist-policy-9"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: blacklistPolicyKeys.target(9),
      exact: true,
    })
    expect(invalidateSpy).not.toHaveBeenCalledWith({
      queryKey: blacklistPolicyKeys.global(),
      exact: true,
    })
  })

  it("treats a Target-local 409 as a terminal conflict: warn, reload, no retry, no success", async () => {
    blacklistPolicyServiceMocks.updateTargetBlacklistPolicy.mockRejectedValue({
      response: {
        status: 409,
        data: { error: { code: "ABORTED" } },
      },
    })

    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false, gcTime: 0 },
        mutations: { retry: 3 },
      },
    })
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateTargetBlacklistPolicy(), { queryClient })

    await act(async () => {
      await expect(result.current.mutateAsync({
        targetId: 9,
        patterns: ["192.0.2.0/24"],
        etag: "stale-etag",
      })).rejects.toBeDefined()
    })

    expect(blacklistPolicyServiceMocks.updateTargetBlacklistPolicy).toHaveBeenCalledTimes(1)
    expect(toastMocks.warning).toHaveBeenCalledWith(
      "pages.settings.blacklist.toast.conflict",
      undefined,
      "update-target-blacklist-policy-9"
    )
    expect(toastMocks.success).not.toHaveBeenCalled()
    expect(toastMocks.errorFromCode).not.toHaveBeenCalled()
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: blacklistPolicyKeys.target(9),
      exact: true,
      refetchType: "active",
    })
  })
})
