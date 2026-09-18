import { render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

const hookMocks = vi.hoisted(() => ({
  operation: {
    operationId: "11111111-1111-4111-8111-111111111111",
    data: {
      status: "succeeded",
      operationId: "11111111-1111-4111-8111-111111111111",
      releaseVersion: "2.0.0",
      migrationStatus: "succeeded",
      cancelledScanCount: 1,
      cancelledTaskCount: 2,
      agentSummary: { expected: 2, ready: 2, missing: 0, unhealthy: 0 },
      createdAt: "2026-09-13T12:00:00Z",
      completedAt: "2026-09-13T12:05:00Z",
    },
  },
  currentVersion: { data: { currentVersion: "2.0.0" } },
}))

vi.mock("next/navigation", () => ({ usePathname: () => "/overview/" }))
vi.mock("@/hooks/use-version", () => ({
  useUpgradeOperation: () => hookMocks.operation,
  useCheckForUpdates: () => hookMocks.currentVersion,
  readPendingSuccessOperationId: () => window.localStorage.getItem("lunafox.upgrade.pendingSuccessOperationId"),
  hasShownUpgradeCompletion: (id: string) => window.localStorage.getItem(`lunafox.upgrade.completionAcknowledged.${id}`) === "1",
  markUpgradeCompletionShown: (id: string) => window.localStorage.setItem(`lunafox.upgrade.completionAcknowledged.${id}`, "1"),
  clearPendingSuccessOperationId: (id: string) => {
    if (window.localStorage.getItem("lunafox.upgrade.pendingSuccessOperationId") === id) window.localStorage.removeItem("lunafox.upgrade.pendingSuccessOperationId")
  },
}))

import { SystemUpgradeCompletionDialog } from "@/components/system-upgrade-completion-dialog"

describe("SystemUpgradeCompletionDialog", () => {
  beforeEach(() => window.localStorage.clear())

  it("shows one verified completion summary for the pending operation", async () => {
    window.localStorage.setItem("lunafox.upgrade.pendingSuccessOperationId", hookMocks.operation.operationId)
    render(<SystemUpgradeCompletionDialog />)

    expect(await screen.findByRole("dialog")).toBeInTheDocument()
    expect(screen.getByText("completion.title")).toBeInTheDocument()
    expect(screen.getByText("completion.versionVerified:{\"version\":\"2.0.0\"}")).toBeInTheDocument()
    await waitFor(() => expect(window.localStorage.getItem(`lunafox.upgrade.completionAcknowledged.${hookMocks.operation.operationId}`)).toBe("1"))
    expect(window.localStorage.getItem("lunafox.upgrade.pendingSuccessOperationId")).toBeNull()
  })
})
