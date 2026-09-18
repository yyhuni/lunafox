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
} from "@/hooks/use-version"

describe("upgrade lifecycle helpers", () => {
  beforeEach(() => {
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
})
