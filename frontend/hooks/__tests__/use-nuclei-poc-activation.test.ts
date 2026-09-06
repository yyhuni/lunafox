import { act, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

const serviceMocks = vi.hoisted(() => ({
  getNucleiPoc: vi.fn(),
  getNucleiPocActiveSyncTaskName: vi.fn(),
  getNucleiPocErrorBody: vi.fn(),
  getNucleiPocErrorReason: vi.fn(),
  getNucleiPocHttpStatus: vi.fn(),
  getNucleiPocSource: vi.fn(),
  getNucleiPocSyncTask: vi.fn(),
  listNucleiPocs: vi.fn(),
  setNucleiPocActivation: vi.fn(),
  syncNucleiPocSource: vi.fn(),
  updateNucleiPoc: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/nuclei-poc.service", () => serviceMocks)
vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

import { nucleiPocKeys, useSetNucleiPocActivation } from "@/hooks/use-nuclei-pocs"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

describe("useSetNucleiPocActivation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    serviceMocks.getNucleiPocHttpStatus.mockImplementation((error: unknown) => {
      if (typeof error === "object" && error !== null && "status" in error) {
        return (error as { status?: number }).status
      }
      return undefined
    })
    serviceMocks.getNucleiPocErrorReason.mockReturnValue(null)
  })

  it("reports the server count and invalidates list/detail projections without refreshing source metadata", async () => {
    serviceMocks.setNucleiPocActivation.mockResolvedValue({ enabled: true, affectedCount: 3 })
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useSetNucleiPocActivation(), { queryClient })

    await act(async () => {
      await result.current.mutateAsync({ enabled: true })
    })

    expect(serviceMocks.setNucleiPocActivation).toHaveBeenCalledTimes(1)
    expect(serviceMocks.setNucleiPocActivation).toHaveBeenCalledWith(
      { enabled: true },
      expect.anything(),
    )
    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.nucleiPoc.activation.success",
      { count: 3 },
      "nuclei-poc-activation-enable",
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: nucleiPocKeys.lists() })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: nucleiPocKeys.details() })
    expect(invalidateSpy).not.toHaveBeenCalledWith({ queryKey: nucleiPocKeys.source() })
  })

  it("treats a zero-count response as a successful no-op", async () => {
    serviceMocks.setNucleiPocActivation.mockResolvedValue({ enabled: false, affectedCount: 0 })
    const { result } = renderHookWithProviders(() => useSetNucleiPocActivation())

    await act(async () => {
      await result.current.mutateAsync({ enabled: false })
    })

    expect(toastMocks.success).toHaveBeenCalledWith(
      "toast.nucleiPoc.activation.noChange",
      { count: 0 },
      "nuclei-poc-activation-disable",
    )
  })

  it("preserves caches for definitive API failures and maps the sync conflict to a warning", async () => {
    const conflict = Object.assign(new Error("sync is running"), { status: 409 })
    serviceMocks.setNucleiPocActivation.mockRejectedValue(conflict)
    serviceMocks.getNucleiPocErrorReason.mockReturnValue("SYNC_ALREADY_RUNNING")
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useSetNucleiPocActivation(), { queryClient })

    await act(async () => {
      await expect(result.current.mutateAsync({ enabled: true })).rejects.toBe(conflict)
    })

    expect(toastMocks.warning).toHaveBeenCalledWith(
      "toast.nucleiPoc.activation.conflict",
      undefined,
      "nuclei-poc-activation-enable",
    )
    expect(invalidateSpy).not.toHaveBeenCalled()
    expect(serviceMocks.setNucleiPocActivation).toHaveBeenCalledTimes(1)
  })

  it("refreshes list/detail once when the transport result is unknown and never retries", async () => {
    const networkError = new Error("connection lost")
    serviceMocks.setNucleiPocActivation.mockRejectedValue(networkError)
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useSetNucleiPocActivation(), { queryClient })

    await act(async () => {
      await expect(result.current.mutateAsync({ enabled: false })).rejects.toBe(networkError)
    })
    await waitFor(() => expect(invalidateSpy).toHaveBeenCalledTimes(2))

    expect(toastMocks.warning).toHaveBeenCalledWith(
      "toast.nucleiPoc.activation.unknown",
      undefined,
      "nuclei-poc-activation-disable",
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: nucleiPocKeys.lists() })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: nucleiPocKeys.details() })
    expect(invalidateSpy).not.toHaveBeenCalledWith({ queryKey: nucleiPocKeys.source() })
    expect(serviceMocks.setNucleiPocActivation).toHaveBeenCalledTimes(1)
  })
})
