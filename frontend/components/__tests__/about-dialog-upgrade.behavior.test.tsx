import { act, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

const apiMocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))
vi.mock("@/lib/api-client", () => ({ api: apiMocks }))

import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { useAboutDialogState } from "@/components/about-dialog-state"
import { AboutDialogVersionInfo } from "@/components/about-dialog-sections"
import type { UpgradeOperation, UpdateCheckResult } from "@/types/version.types"

const digest = `sha256:${"a".repeat(64)}`
const candidate: NonNullable<UpdateCheckResult["candidate"]> = {
  name: "releaseManifests/release-1.1.0",
  manifestId: "release-1.1.0",
  manifestDigest: digest,
  releaseVersion: "1.1.0",
  deploymentMode: "single-node-compose",
  compatibilityRange: ">=1.0.0 <2.0.0",
  maintenanceWindowMinutes: 15,
  requiresAdminConfirmation: true,
  databaseMigration: { hasDatabaseMigration: false, migrationType: "none", policyVersion: 1 },
  runtimeImageDigests: { server: digest, frontend: digest, nginx: digest },
  engineDigests: [digest],
}

const updateResult: UpdateCheckResult = {
  currentVersion: "1.0.0",
  hasUpdate: true,
  eligible: true,
  candidate,
}

const operation: UpgradeOperation = {
  name: "upgradeOperations/11111111-1111-4111-8111-111111111111",
  operationId: "11111111-1111-4111-8111-111111111111",
  requestId: "22222222-2222-4222-8222-222222222222",
  operatorId: 7,
  manifestId: candidate.manifestId,
  manifestDigest: digest,
  releaseVersion: candidate.releaseVersion,
  compatibilityRange: candidate.compatibilityRange,
  maintenanceWindowMinutes: candidate.maintenanceWindowMinutes,
  status: "queued",
  migrationStatus: "not_started",
  migrationType: "none",
  cancelledScanCount: 2,
  cancelledTaskCount: 3,
  agentSummary: { expected: 0, ready: 0, missing: 0, unhealthy: 0 },
  observedDigests: {},
  stageTimes: { queued: "2026-09-13T12:00:00Z" },
  createdAt: "2026-09-13T12:00:00Z",
  updatedAt: "2026-09-13T12:00:00Z",
  completedAt: null,
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

describe("about dialog upgrade behavior", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    window.localStorage.clear()
  })

  it("requires the confirmation flow and creates one durable operation for same-tick clicks", async () => {
    const createResponse = deferred<{ data: UpgradeOperation }>()
    apiMocks.get.mockResolvedValue({ data: operation })
    apiMocks.post.mockImplementation((path: string) => {
      if (path === "/system:checkForUpdates") return Promise.resolve({ data: updateResult })
      if (path === "/upgradeOperations") return createResponse.promise
      throw new Error(`unexpected POST ${path}`)
    })

    const { result } = renderHookWithProviders(() => useAboutDialogState())
    await act(async () => {
      await result.current.handleCheckUpdate()
    })
    act(() => result.current.handleStartUpgrade())
    expect(result.current.confirmOpen).toBe(true)

    act(() => {
      result.current.handleConfirmUpgrade(true)
      result.current.handleConfirmUpgrade(true)
    })

		await waitFor(() => expect(apiMocks.post.mock.calls.filter(([path]) => path === "/upgradeOperations")).toHaveLength(1))
    const request = apiMocks.post.mock.calls.find(([path]) => path === "/upgradeOperations")?.[1] as Record<string, unknown>
    expect(request).toMatchObject({ manifestId: candidate.manifestId, manifestDigest: digest, confirmed: true })

    await act(async () => {
      createResponse.resolve({ data: operation })
      await createResponse.promise
    })
    await waitFor(() => expect(window.localStorage.getItem("lunafox.upgrade.operationId")).toBe(operation.operationId))
    expect(result.current.operation.operationId).toBe(operation.operationId)
  })

  it("uses the persisted operation id after a fresh mount", async () => {
    window.localStorage.setItem("lunafox.upgrade.operationId", operation.operationId)
    apiMocks.get.mockResolvedValue({ data: operation })

    const { result } = renderHookWithProviders(() => useAboutDialogState())
    await waitFor(() => expect(result.current.operation.data?.operationId).toBe(operation.operationId))
    expect(apiMocks.get).toHaveBeenCalledWith(`/upgradeOperations/${operation.operationId}`)
    expect(result.current.operation.lastConfirmedStage).toBe("queued")
  })

  it("does not poll while the about entry is closed, then restores the persisted operation when opened", async () => {
    window.localStorage.setItem("lunafox.upgrade.operationId", operation.operationId)
    apiMocks.get.mockResolvedValue({ data: operation })

    const { result, rerender } = renderHookWithProviders(
      ({ enabled }: { enabled: boolean }) => useAboutDialogState({ enabled }),
      { initialProps: { enabled: false } },
    )
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(apiMocks.get).not.toHaveBeenCalled()

    rerender({ enabled: true })
    await waitFor(() => expect(result.current.operation.data?.operationId).toBe(operation.operationId))
    expect(apiMocks.get).toHaveBeenCalledWith(`/upgradeOperations/${operation.operationId}`)
  })

  it("uses the server operation as the source of truth after reconnect", async () => {
    window.localStorage.setItem("lunafox.upgrade.operationId", operation.operationId)
    const recovered = {
      ...operation,
      status: "restarting" as const,
      stageTimes: { ...operation.stageTimes, restarting: "2026-09-13T12:05:00Z" },
    }
    apiMocks.get.mockResolvedValue({ data: recovered })

    const { result } = renderHookWithProviders(() => useAboutDialogState())
    await waitFor(() => expect(result.current.operation.data?.status).toBe("restarting"))
    expect(result.current.operation.lastConfirmedStage).toBe("restarting")
    expect(result.current.checkError).toBeNull()
  })

  it("does not create an operation until the confirmation checkbox is acknowledged", async () => {
    apiMocks.get.mockResolvedValue({ data: operation })
    apiMocks.post.mockImplementation((path: string) => {
      if (path === "/system:checkForUpdates") return Promise.resolve({ data: updateResult })
      if (path === "/upgradeOperations") return Promise.resolve({ data: operation })
      throw new Error(`unexpected POST ${path}`)
    })

    const { result } = renderHookWithProviders(() => useAboutDialogState())
    await act(async () => {
      await result.current.handleCheckUpdate()
    })
    act(() => result.current.handleStartUpgrade())
    act(() => result.current.handleConfirmUpgrade(false))
    expect(apiMocks.post.mock.calls.filter(([path]) => path === "/upgradeOperations")).toHaveLength(0)

    act(() => result.current.handleConfirmUpgrade(true))
    await waitFor(() => expect(apiMocks.post.mock.calls.filter(([path]) => path === "/upgradeOperations")).toHaveLength(1))
  })

  it("exposes a reconnecting state without treating it as a terminal failure", () => {
    const retry = vi.fn()
    render(
      <AboutDialogVersionInfo
        t={(key) => key}
        currentVersion="1.0.0"
        candidate={candidate}
        hasUpdate={false}
        checkError={null}
        isChecking={false}
        isCreating={false}
        operation={{ data: { ...operation, status: "needs_attention" }, isReconnecting: true }}
        onCheckUpdate={vi.fn()}
        onStartUpgrade={vi.fn()}
        onRetry={retry}
      />,
    )
    expect(screen.getByText("reconnecting")).toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: "retryUpgrade" }))
    expect(retry).toHaveBeenCalledTimes(1)
  })
})
