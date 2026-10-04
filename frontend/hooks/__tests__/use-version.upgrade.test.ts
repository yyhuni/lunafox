import { act, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import {
  clearPendingSuccessOperationId,
  clearStoredUpgradeOperationId,
  hasShownUpgradeCompletion,
  isUpgradeOperationActive,
  markUpgradeCompletionShown,
  persistPendingSuccessOperationId,
  persistUpgradeOperationId,
  readPendingSuccessOperationId,
  readStoredUpgradeOperationId,
  upgradeOperationPollDelay,
  upgradeUserStageForStatus,
  useUpgradeOperation,
} from "@/hooks/use-version"
import { VersionService } from "@/services/version.service"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import type { UpgradeOperationFull } from "@/types/version.types"

describe("upgrade lifecycle helpers", () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    window.localStorage.clear()
  })

  it("persists and clears the accepted operation without clearing a newer operation", () => {
    persistUpgradeOperationId("old-operation")
    persistUpgradeOperationId("new-operation")

    clearStoredUpgradeOperationId("old-operation")
    expect(readStoredUpgradeOperationId()).toBe("new-operation")

    clearStoredUpgradeOperationId("new-operation")
    expect(readStoredUpgradeOperationId()).toBeNull()
  })

  it("emits same-tab operation changes so the route gate can recover immediately", () => {
    const listener = vi.fn()
    window.addEventListener("lunafox:upgrade-operation-changed", listener)

    persistUpgradeOperationId("operation-id")
    clearStoredUpgradeOperationId("operation-id")

    expect(listener).toHaveBeenCalledTimes(2)
    window.removeEventListener("lunafox:upgrade-operation-changed", listener)
  })

  it("maps server states to stable user stages and treats only non-terminal states as active", () => {
    expect(upgradeUserStageForStatus("queued")).toBe("preparing")
    expect(upgradeUserStageForStatus("migrating")).toBe("updating")
    expect(upgradeUserStageForStatus("restarting")).toBe("restarting")
    expect(upgradeUserStageForStatus("needs_attention")).toBe("finished")
    expect(isUpgradeOperationActive("verifying")).toBe(true)
    expect(isUpgradeOperationActive("succeeded")).toBe(false)
    expect(isUpgradeOperationActive("needs_recovery")).toBe(false)
  })

  it("acknowledges completion per operation and keeps the pending success scoped", () => {
    persistPendingSuccessOperationId("operation-id")
    expect(readPendingSuccessOperationId()).toBe("operation-id")
    expect(hasShownUpgradeCompletion("operation-id")).toBe(false)

    markUpgradeCompletionShown("operation-id")
    expect(hasShownUpgradeCompletion("operation-id")).toBe(true)
    expect(hasShownUpgradeCompletion("other-operation")).toBe(false)

    clearPendingSuccessOperationId("other-operation")
    expect(readPendingSuccessOperationId()).toBe("operation-id")
    clearPendingSuccessOperationId("operation-id")
    expect(readPendingSuccessOperationId()).toBeNull()
  })

  it("uses bounded polling backoff for reconnects", () => {
    expect(upgradeOperationPollDelay(0)).toBe(1500)
    expect(upgradeOperationPollDelay(3)).toBe(12000)
    expect(upgradeOperationPollDelay(99)).toBe(15000)
  })

  it("keeps automatic recovery on the active endpoint after a stale hint expires", async () => {
    const staleOperationId = "11111111-1111-4111-8111-111111111111"
    persistUpgradeOperationId(staleOperationId)
    const activeLookup = vi.spyOn(VersionService, "getActiveUpgradeOperationFull").mockResolvedValue(null)
    const staleLookup = vi.spyOn(VersionService, "getUpgradeOperationFull").mockRejectedValue(new Error("stale lookup must not run"))

    const { result } = renderHookWithProviders(() => useUpgradeOperation())
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(activeLookup).toHaveBeenCalledTimes(1)
    expect(staleLookup).not.toHaveBeenCalled()
    expect(result.current.data).toBeUndefined()
    expect(result.current.operationId).toBeNull()
    expect(readStoredUpgradeOperationId()).toBeNull()
    await new Promise((resolve) => setTimeout(resolve, 1_700))
    expect(activeLookup).toHaveBeenCalledTimes(1)
  }, 15_000)

  it("keeps a persisted hint while the active view is resolving, then releases it when empty", async () => {
    const staleOperationId = "11111111-1111-4111-8111-111111111111"
    persistUpgradeOperationId(staleOperationId)
    let resolveActive!: (value: null) => void
    const activePromise = new Promise<null>((resolve) => { resolveActive = resolve })
    const activeLookup = vi.spyOn(VersionService, "getActiveUpgradeOperationFull").mockReturnValue(activePromise)
    const staleLookup = vi.spyOn(VersionService, "getUpgradeOperationFull").mockRejectedValue(new Error("stale lookup must not run"))

    const { result } = renderHookWithProviders(() => useUpgradeOperation())
    await waitFor(() => expect(activeLookup).toHaveBeenCalledTimes(1))
    expect(result.current.operationId).toBe(staleOperationId)
    expect(result.current.isResolving).toBe(true)
    expect(staleLookup).not.toHaveBeenCalled()

    await act(async () => {
      resolveActive(null)
      await activePromise
    })
    await waitFor(() => expect(result.current.operationId).toBeNull())
    expect(readStoredUpgradeOperationId()).toBeNull()
  })

  it("does not clear a newer storage hint when the active request finishes empty", async () => {
    const initialOperationId = "11111111-1111-4111-8111-111111111111"
    const newerOperationId = "22222222-2222-4222-8222-222222222222"
    persistUpgradeOperationId(initialOperationId)
    let resolveActive!: (value: null) => void
    const activePromise = new Promise<null>((resolve) => { resolveActive = resolve })
    const activeLookup = vi.spyOn(VersionService, "getActiveUpgradeOperationFull").mockReturnValue(activePromise)

    const { result } = renderHookWithProviders(() => useUpgradeOperation())
    await waitFor(() => expect(activeLookup).toHaveBeenCalledTimes(1))
    window.localStorage.setItem("lunafox.upgrade.operationId", newerOperationId)

    await act(async () => {
      resolveActive(null)
      await activePromise
    })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.operationId).toBeNull()
    expect(readStoredUpgradeOperationId()).toBe(newerOperationId)
  })

  it("refreshes the fixed active query when a new operation hint is persisted", async () => {
    const operation = {
      operationId: "33333333-3333-4333-8333-333333333333",
      status: "updating",
    } as unknown as UpgradeOperationFull
    const activeLookup = vi.spyOn(VersionService, "getActiveUpgradeOperationFull")
      .mockResolvedValueOnce(null)
      .mockResolvedValueOnce(operation)

    const { result } = renderHookWithProviders(() => useUpgradeOperation())
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    act(() => persistUpgradeOperationId(operation.operationId))
    await waitFor(() => expect(result.current.data?.operationId).toBe(operation.operationId))
    expect(activeLookup).toHaveBeenCalledTimes(2)
  })

  it("keeps the new hint fail-closed while an empty active cache refetches", async () => {
    const operation = {
      operationId: "44444444-4444-4444-8444-444444444444",
      status: "updating",
    } as unknown as UpgradeOperationFull
    let resolveActive!: (value: UpgradeOperationFull) => void
    const activePromise = new Promise<UpgradeOperationFull>((resolve) => { resolveActive = resolve })
    const activeLookup = vi.spyOn(VersionService, "getActiveUpgradeOperationFull")
      .mockResolvedValueOnce(null)
      .mockReturnValueOnce(activePromise)

    const { result } = renderHookWithProviders(() => useUpgradeOperation())
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    act(() => persistUpgradeOperationId(operation.operationId))
    await waitFor(() => expect(activeLookup).toHaveBeenCalledTimes(2))
    expect(result.current.operationId).toBe(operation.operationId)
    expect(result.current.isResolving).toBe(true)

    await act(async () => {
      resolveActive(operation)
      await activePromise
    })
    await waitFor(() => expect(result.current.data?.operationId).toBe(operation.operationId))
  })
})
