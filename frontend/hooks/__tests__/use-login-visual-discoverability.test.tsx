import { act, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const serviceMocks = vi.hoisted(() => ({
  checkDiscoverability: vi.fn(),
  unlockDiscoverability: vi.fn(),
}))
const localUnlockMocks = vi.hoisted(() => ({
  unlocked: false,
  unlock: vi.fn(() => {
    if (localUnlockMocks.unlocked) return false
    localUnlockMocks.unlocked = true
    return true
  }),
}))

vi.mock("@/services/login-visual.service", () => ({
  LoginVisualService: serviceMocks,
}))

vi.mock("@/lib/login-visual-unlock", () => ({
  isLoginVisualUnlocked: () => localUnlockMocks.unlocked,
  unlockLoginVisual: localUnlockMocks.unlock,
  useLoginVisualUnlocked: () => localUnlockMocks.unlocked,
}))

import {
  loginVisualKeys,
  useLoginVisualDiscoverability,
  useUnlockLoginVisualDiscoverability,
} from "@/hooks/use-login-visual"

describe("login visual discoverability hooks", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localUnlockMocks.unlocked = false
  })

  it("uses the account result after it loads instead of a stale browser fallback", async () => {
    localUnlockMocks.unlocked = true
    serviceMocks.checkDiscoverability.mockResolvedValue({ unlocked: false })

    const { result } = renderHookWithProviders(() => useLoginVisualDiscoverability())

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.unlocked).toBe(false)
  })

  it("hydrates the local continuity cache from an unlocked account", async () => {
    serviceMocks.checkDiscoverability.mockResolvedValue({ unlocked: true })

    const { result } = renderHookWithProviders(() => useLoginVisualDiscoverability())

    await waitFor(() => expect(result.current.unlocked).toBe(true))
    expect(localUnlockMocks.unlock).toHaveBeenCalledTimes(1)
  })

  it("writes the account unlock result into the discoverability cache", async () => {
    serviceMocks.unlockDiscoverability.mockResolvedValue({ unlocked: true })
    const queryClient = createTestQueryClient()
    // The mutation writes an otherwise unobserved entry; retain it for the cache assertion.
    queryClient.setQueryDefaults(loginVisualKeys.discoverability(), { gcTime: Infinity })
    const { result } = renderHookWithProviders(() => useUnlockLoginVisualDiscoverability(), { queryClient })

    await act(async () => {
      await result.current.mutateAsync()
    })

    expect(queryClient.getQueryData(loginVisualKeys.discoverability())).toEqual({ unlocked: true })
    expect(localUnlockMocks.unlock).toHaveBeenCalledTimes(1)
  })
})
