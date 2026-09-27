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
import type { UpgradeOperationFull } from "@/types/version.types"

const digest = `sha256:${"a".repeat(64)}`
const FULL_ONLY_OPERATION_FIELDS = new Set(["executionMode", "workDisposition", "planSummary", "confirmedDeploymentVersion", "agentDiagnostics", "hostActivity"])

function toBasicOperation(operation: UpgradeOperationFull): Record<string, unknown> {
  return Object.fromEntries(Object.entries(operation).filter(([key]) => !FULL_ONLY_OPERATION_FIELDS.has(key)))
}

function makeOperation(status: UpgradeOperationFull["status"]): UpgradeOperationFull {
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
    executionMode: "full",
    workDisposition: "cancelled",
    planSummary: { touchedServices: ["agent", "bootstrap", "engine", "engine_package", "engine_runtime", "frontend", "migration", "nginx", "server"] },
    confirmedDeploymentVersion: status === "succeeded" ? "1.1.0" : "1.0.0",
    agentDiagnostics: [],
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
    apiMocks.post.mockResolvedValue({ data: toBasicOperation(retried) })

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

  it("renders safe long-stage events through the shared followable text stream", async () => {
    const operation = makeOperation("updating")
    operation.logs = [
      ...operation.logs,
      {
        timestamp: "2026-09-13T12:02:00Z",
        level: "info",
        stage: "updating",
        messageKey: "pullImagesStarted",
        message: "Pulling release images",
      },
      {
        timestamp: "2026-09-13T12:03:00Z",
        level: "info",
        stage: "updating",
        messageKey: "servicesUpdateStarted",
        message: "Updating core services",
      },
    ]
    window.localStorage.setItem("lunafox.upgrade.operationId", operation.operationId)
    apiMocks.get.mockResolvedValue({ data: operation })

    const { container } = renderWithProviders(<SystemUpgradeStatus />)

    expect(await screen.findByTestId("system-upgrade-status")).toBeInTheDocument()
    const viewer = container.querySelector('[data-slot="raw-log-viewer"]')
    expect(viewer).toHaveTextContent("Pulling release images")
    expect(viewer).toHaveTextContent("Updating core services")
    expect(screen.getByRole("button", { name: "logs.copy" })).toBeEnabled()
  })

  it("renders a current host activity fact outside the event log and clears it when terminal", async () => {
    const active = makeOperation("updating")
    active.stageTimes.updating = "2026-09-13T12:01:00Z"
    active.hostActivity = {
      action: "pull_images",
      startedAt: "2026-09-13T12:01:00Z",
      lastHeartbeatAt: "2026-09-13T12:04:00Z",
    }
    window.localStorage.setItem("lunafox.upgrade.operationId", active.operationId)
    const terminal = { ...makeOperation("succeeded"), hostActivity: null }
    apiMocks.get.mockResolvedValueOnce({ data: active }).mockResolvedValueOnce({ data: terminal })

    const { container } = renderWithProviders(<SystemUpgradeStatus />)

    const activity = await screen.findByTestId("system-upgrade-host-activity")
    expect(activity).toHaveTextContent("hostActivity.title")
    expect(activity).toHaveTextContent("hostActivity.actions.pull_images")
    expect(activity).toHaveTextContent("hostActivity.waiting")
    const logViewer = container.querySelector('[data-slot="raw-log-viewer"]')
    expect(logViewer).not.toHaveTextContent("hostActivity.actions.pull_images")
    expect(active.logs.some((entry) => /heartbeat/i.test(entry.messageKey))).toBe(false)

    fireEvent.click(screen.getByRole("button", { name: "actions.refresh" }))
    await waitFor(() => expect(screen.queryByTestId("system-upgrade-host-activity")).not.toBeInTheDocument())
  })

  it("renders blocking Agent evidence and owns the route scroll viewport", async () => {
    const operation = makeOperation("agent_verifying")
    operation.agentSummary = { expected: 2, ready: 1, missing: 1, unhealthy: 0 }
    operation.agentDiagnostics = [{
      agentId: 9,
      name: "agents/9",
      displayNameSnapshot: "edge-9",
      reasonCode: "heartbeat_missing_or_stale",
      detail: "Agent heartbeat is missing or stale",
      source: "server_observation",
    }]
    window.localStorage.setItem("lunafox.upgrade.operationId", operation.operationId)
    apiMocks.get.mockResolvedValue({ data: operation })

    const { container } = renderWithProviders(<SystemUpgradeStatus />)

    const route = await screen.findByTestId("system-upgrade-status")
    expect(route).toHaveClass("h-svh", "overflow-y-auto")
    expect(route).toHaveAttribute("tabindex", "0")
    expect(screen.getByTestId("system-upgrade-agent-diagnostics")).toBeInTheDocument()
    expect(screen.getByText("edge-9")).toBeInTheDocument()
    expect(screen.getByText("agents/9")).toBeInTheDocument()
    expect(screen.getByText("Agent heartbeat is missing or stale")).toBeInTheDocument()
    expect(container.querySelector('[data-slot="raw-log-viewer"]')).toBeInTheDocument()
  })

  it("replaces recovered diagnostics and retains the last confirmed set during reconnect", async () => {
    const first = makeOperation("agent_verifying")
    first.agentDiagnostics = [{
      agentId: 9,
      name: "agents/9",
      displayNameSnapshot: "edge-9",
      reasonCode: "heartbeat_missing_or_stale",
      detail: "Agent heartbeat is missing or stale",
      source: "server_observation",
    }]
    const recovered = {
      ...first,
      agentDiagnostics: [{
        agentId: 12,
        name: "agents/12",
        reasonCode: "version_mismatch",
        detail: "Agent is running a different version",
        source: "server_observation" as const,
      }],
    }
    window.localStorage.setItem("lunafox.upgrade.operationId", first.operationId)
    let poll = 0
    apiMocks.get.mockImplementation(() => {
      poll += 1
      if (poll === 1) return Promise.resolve({ data: first })
      if (poll === 2) return Promise.resolve({ data: recovered })
      return Promise.reject(new TypeError("network down"))
    })

    renderWithProviders(<SystemUpgradeStatus />)
    await screen.findByTestId("system-upgrade-status")
    expect(screen.getByText("edge-9")).toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", { name: "actions.refresh" }))
    await waitFor(() => expect(screen.getAllByText("agents/12").length).toBeGreaterThan(0))
    expect(screen.queryByText("edge-9")).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", { name: "actions.refresh" }))
    await waitFor(() => expect(apiMocks.get).toHaveBeenCalledTimes(3))
    expect(screen.getAllByText("agents/12").length).toBeGreaterThan(0)
  })

  it("renders a frontend-only timeline without stopping, migration, cancellation, or Agent claims", async () => {
    const operation = {
      ...makeOperation("restarting"),
      executionMode: "frontend_only" as const,
      workDisposition: "not_required" as const,
      planSummary: { touchedServices: ["frontend"] },
      stageTimes: {
        queued: "2026-09-13T12:00:00Z",
        preflight: "2026-09-13T12:01:00Z",
        updating: "2026-09-13T12:02:00Z",
        restarting: "2026-09-13T12:03:00Z",
      },
      agentDiagnostics: [{
        agentId: 9,
        name: "agents/9",
        reasonCode: "legacy_unknown",
        detail: "Historical detail",
        source: "historical" as const,
      }],
    }
    window.localStorage.setItem("lunafox.upgrade.operationId", operation.operationId)
    apiMocks.get.mockResolvedValue({ data: operation })

    renderWithProviders(<SystemUpgradeStatus />)

    expect(await screen.findByTestId("system-upgrade-status")).toBeInTheDocument()
    expect(screen.getByTestId("system-upgrade-frontend-only-scope")).toBeInTheDocument()
    const stages = screen.getByLabelText("timeline.stageList")
    expect(stages.querySelector('[data-stage="stopping"]')).toBeNull()
    expect(stages.querySelector('[data-stage="preparing"]')).toBeInTheDocument()
    expect(stages.querySelector('[data-stage="restarting"]')).toHaveAttribute("data-stage-state", "current")
    expect(screen.queryByText("facts.migration")).not.toBeInTheDocument()
    expect(screen.queryByText("facts.cancelledWork")).not.toBeInTheDocument()
    expect(screen.queryByText("facts.agents")).not.toBeInTheDocument()
    expect(screen.queryByTestId("system-upgrade-agent-diagnostics")).not.toBeInTheDocument()
  })
})
