import { fireEvent, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

const apiMocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))
vi.mock("@/lib/api-client", () => ({ api: apiMocks }))

const navigationMocks = vi.hoisted(() => ({ replace: vi.fn() }))
vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: navigationMocks.replace }),
  usePathname: () => "/system-upgrade/",
}))

import { renderWithProviders } from "@/test/utils/render-with-providers"
import { SystemUpgradeStatus } from "@/components/system-upgrade-status"
import type { UpgradeOperation } from "@/types/version.types"

const digest = `sha256:${"a".repeat(64)}`

function makeOperation(status: UpgradeOperation["status"]): UpgradeOperation {
  return {
    name: "upgradeOperations/11111111-1111-4111-8111-111111111111",
    operationId: "11111111-1111-4111-8111-111111111111",
    requestId: "22222222-2222-4222-8222-222222222222",
    operatorId: 7,
    manifestId: "release-1.1.0",
    manifestDigest: digest,
    currentVersion: "1.0.0",
    releaseVersion: "1.1.0",
    compatibilityRange: ">=1.0.0 <2.0.0",
    maintenanceWindowMinutes: 15,
    status,
    migrationStatus: status === "migrating" ? "running" : status === "succeeded" ? "succeeded" : "not_started",
    migrationType: "none",
    cancelledScanCount: 2,
    cancelledTaskCount: 3,
    agentSummary: { expected: 2, ready: status === "succeeded" ? 2 : 0, missing: 0, unhealthy: 0 },
    observedDigests: {},
    logs: [{ timestamp: "2026-09-13T12:00:00Z", level: "info", stage: "queued", messageKey: "requestAccepted", message: "Upgrade request accepted" }],
    diagnostic: status === "failed" ? "digest verification failed" : undefined,
    stageTimes: { queued: "2026-09-13T12:00:00Z", ...(status !== "queued" ? { stopping: "2026-09-13T12:01:00Z" } : {}) },
    createdAt: "2026-09-13T12:00:00Z",
    updatedAt: "2026-09-13T12:05:00Z",
    completedAt: status === "succeeded" || status === "failed" ? "2026-09-13T12:05:00Z" : null,
  }
}

describe("system upgrade status", () => {
  beforeEach(() => {
    window.localStorage.clear()
    apiMocks.get.mockReset()
    apiMocks.post.mockReset()
    navigationMocks.replace.mockReset()
  })

  it("renders server-confirmed stages and facts without a fake remaining-time estimate", async () => {
    const operation = makeOperation("restarting")
    window.localStorage.setItem("lunafox.upgrade.operationId", operation.operationId)
    apiMocks.get.mockResolvedValue({ data: operation })

    renderWithProviders(<SystemUpgradeStatus />)

    expect(await screen.findByTestId("system-upgrade-status")).toBeInTheDocument()
    expect(screen.getByTestId("system-upgrade-status-badge")).toHaveTextContent("status.restarting")
    expect(screen.getByRole("progressbar")).toBeInTheDocument()
    expect(screen.getByText(/facts\.operationId/)).toBeInTheDocument()
    expect(screen.queryByText(/remaining|剩余|ETA/i)).not.toBeInTheDocument()
    expect(screen.getByLabelText("timeline.stageList").querySelector('[data-stage="restarting"]')).toHaveAttribute("data-stage-state", "current")
  })

  it("offers retry only for a failed operation and protects the request while pending", async () => {
    const failed = makeOperation("failed")
    const retried = makeOperation("queued")
    window.localStorage.setItem("lunafox.upgrade.operationId", failed.operationId)
    apiMocks.get.mockResolvedValue({ data: failed })
    apiMocks.post.mockResolvedValue({ data: retried })

    renderWithProviders(<SystemUpgradeStatus />)
    await screen.findByTestId("system-upgrade-status")

    const retryButton = screen.getByRole("button", { name: "actions.retry" })
    fireEvent.click(retryButton)
    await waitFor(() => expect(apiMocks.post).toHaveBeenCalledTimes(1))
    fireEvent.click(retryButton)
    expect(apiMocks.post).toHaveBeenCalledTimes(1)
    expect(apiMocks.post).toHaveBeenCalledWith(`/upgradeOperations/${failed.operationId}:retry`, { confirmed: true })
    await waitFor(() => expect(screen.getByTestId("system-upgrade-status-badge")).toHaveTextContent("status.queued"))
  })

  it("does not mark phases after a stopped checkpoint as complete", async () => {
    const attention = makeOperation("needs_attention")
    window.localStorage.setItem("lunafox.upgrade.operationId", attention.operationId)
    apiMocks.get.mockResolvedValue({ data: attention })

    renderWithProviders(<SystemUpgradeStatus />)
    await screen.findByTestId("system-upgrade-status")

    const stages = screen.getByLabelText("timeline.stageList")
    expect(stages.querySelector('[data-stage="preparing"]')).toHaveAttribute("data-stage-state", "complete")
    expect(stages.querySelector('[data-stage="stopping"]')).toHaveAttribute("data-stage-state", "complete")
    expect(stages.querySelector('[data-stage="updating"]')).toHaveAttribute("data-stage-state", "pending")
    expect(stages.querySelector('[data-stage="restarting"]')).toHaveAttribute("data-stage-state", "pending")
    expect(stages.querySelector('[data-stage="finished"]')).toHaveAttribute("data-stage-state", "current")
  })
})
