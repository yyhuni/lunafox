import { act, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

const apiMocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))
vi.mock("@/lib/api-client", () => ({ api: apiMocks }))

import { renderHookWithProviders, renderWithProviders } from "@/test/utils/render-with-providers"
import { AboutDialog } from "@/components/about-dialog"
import { useAboutDialogState } from "@/components/about-dialog-state"
import { AboutDialogVersionInfo } from "@/components/about-dialog-sections"
import { SidebarMenuButton, SidebarProvider } from "@/components/ui/sidebar"
import type { UpgradeOperation, UpgradeOperationFull, UpdateCheckResult } from "@/types/version.types"

const digest = `sha256:${"a".repeat(64)}`
const releaseNotesBody = "## English\n\n- Test release notes.\n\n## 简体中文\n\n- 测试发布说明。\n"
const releaseNotesSha256 = "sha256:4406112ce062dd05feacce5f519b8cb7250fd01c7237c43e0f7335da912e8188"
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
  releaseNotes: { body: releaseNotesBody, sha256: releaseNotesSha256 },
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
  currentVersion: "1.0.0",
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
  logs: [{ timestamp: "2026-09-13T12:00:00Z", level: "info", stage: "queued", messageKey: "requestAccepted", message: "Upgrade request accepted" }],
  stageTimes: { queued: "2026-09-13T12:00:00Z" },
  createdAt: "2026-09-13T12:00:00Z",
  updatedAt: "2026-09-13T12:00:00Z",
  completedAt: null,
}
const fullOperation: UpgradeOperationFull = {
  ...operation,
  executionMode: "full",
  workDisposition: "cancelled",
  planSummary: { touchedServices: ["agent", "bootstrap", "engine", "engine_package", "engine_runtime", "frontend", "migration", "nginx", "server"] },
  confirmedDeploymentVersion: operation.currentVersion,
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
    apiMocks.get.mockResolvedValue({ data: fullOperation })
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
    apiMocks.get.mockImplementation((path: string) => {
      if (path === "/upgradeOperations:active") {
        return Promise.reject({
          isAxiosError: true,
          response: { status: 404, data: { error: { code: "NOT_FOUND", message: "No active upgrade operation." } } },
          message: "not found",
        })
      }
      return Promise.resolve({ data: fullOperation })
    })

    const { result } = renderHookWithProviders(() => useAboutDialogState())
    await waitFor(() => expect(result.current.operation.data?.operationId).toBe(operation.operationId))
    expect(apiMocks.get).toHaveBeenCalledWith("/upgradeOperations:active", { params: { view: "FULL" } })
    expect(apiMocks.get).toHaveBeenCalledWith(`/upgradeOperations/${operation.operationId}`, { params: { view: "FULL" } })
    expect(result.current.operation.lastConfirmedStage).toBe("queued")
  })

  it("does not poll while the about entry is closed, then restores the persisted operation when opened", async () => {
    window.localStorage.setItem("lunafox.upgrade.operationId", operation.operationId)
    apiMocks.get.mockImplementation((path: string) => {
      if (path === "/upgradeOperations:active") {
        return Promise.reject({
          isAxiosError: true,
          response: { status: 404, data: { error: { code: "NOT_FOUND", message: "No active upgrade operation." } } },
          message: "not found",
        })
      }
      return Promise.resolve({ data: fullOperation })
    })

    const { result, rerender } = renderHookWithProviders(
      ({ enabled }: { enabled: boolean }) => useAboutDialogState({ enabled }),
      { initialProps: { enabled: false } },
    )
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(apiMocks.get).not.toHaveBeenCalled()

    rerender({ enabled: true })
    await waitFor(() => expect(result.current.operation.data?.operationId).toBe(operation.operationId))
    expect(apiMocks.get).toHaveBeenCalledWith("/upgradeOperations:active", { params: { view: "FULL" } })
    expect(apiMocks.get).toHaveBeenCalledWith(`/upgradeOperations/${operation.operationId}`, { params: { view: "FULL" } })
  })

  it("uses the server operation as the source of truth after reconnect", async () => {
    window.localStorage.setItem("lunafox.upgrade.operationId", operation.operationId)
    const recovered = {
      ...fullOperation,
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
    apiMocks.get.mockResolvedValue({ data: fullOperation })
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
    const viewStatus = vi.fn()
    render(
      <AboutDialogVersionInfo
        t={(key) => key}
        currentVersion="1.0.0"
        candidate={candidate}
        hasUpdate={false}
        checkError={null}
        isChecking={false}
        isCreating={false}
        canStartUpgrade={false}
        operation={{ data: { ...fullOperation, status: "needs_attention" }, isReconnecting: true }}
        onCheckUpdate={vi.fn()}
        onStartUpgrade={vi.fn()}
        onRetry={retry}
        onViewStatus={viewStatus}
      />,
    )
    expect(screen.getByText("reconnecting")).toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: "viewUpgradeStatus" }))
    expect(viewStatus).toHaveBeenCalledTimes(1)
    expect(retry).not.toHaveBeenCalled()
  })

  it("does not claim cancellation for a frontend-only FULL operation", () => {
    render(
      <AboutDialogVersionInfo
        t={(key) => key}
        currentVersion="1.0.0"
        candidate={candidate}
        hasUpdate={false}
        checkError={null}
        isChecking={false}
        isCreating={false}
        canStartUpgrade={false}
        operation={{
          data: {
            ...fullOperation,
            executionMode: "frontend_only",
            workDisposition: "not_required",
            planSummary: { touchedServices: ["frontend"] },
          },
        }}
        onCheckUpdate={vi.fn()}
        onStartUpgrade={vi.fn()}
        onRetry={vi.fn()}
      />,
    )

    expect(screen.getByText("frontendOnlyScope")).toBeInTheDocument()
    expect(screen.queryByText(/cancelledWork/)).not.toBeInTheDocument()
  })

  it("does not show upgrade-only details when the candidate is already installed", () => {
    render(
      <AboutDialogVersionInfo
        t={(key) => key}
        currentVersion={candidate.releaseVersion}
        candidate={candidate}
        hasUpdate={false}
        checkError={null}
        isChecking={false}
        isCreating={false}
        canStartUpgrade={false}
        operation={{}}
        onCheckUpdate={vi.fn()}
        onStartUpgrade={vi.fn()}
        onRetry={vi.fn()}
      />,
    )

    expect(screen.getByText("upToDate")).toBeInTheDocument()
    expect(screen.queryByText("manifestDigest")).not.toBeInTheDocument()
    expect(screen.queryByText("maintenanceWindow")).not.toBeInTheDocument()
    expect(screen.queryByText("noMigration")).not.toBeInTheDocument()
  })

  it("shows verified release notes as inert text and links to the candidate release", () => {
    render(
      <AboutDialogVersionInfo
        t={(key) => key}
        githubRepo="https://github.com/yyhuni/lunafox/"
        currentVersion="1.0.0"
        candidate={candidate}
        hasUpdate
        checkError={null}
        isChecking={false}
        isCreating={false}
        canStartUpgrade={false}
        operation={{}}
        onCheckUpdate={vi.fn()}
        onStartUpgrade={vi.fn()}
        onRetry={vi.fn()}
      />,
    )

    expect(screen.getByTestId("release-notes-body")).toHaveTextContent("Test release notes.")
    expect(screen.getByTestId("release-notes-body")).toHaveTextContent("测试发布说明。")
    const releaseLink = screen.getByRole("link", { name: "viewRelease" })
    expect(releaseLink).toHaveAttribute("href", "https://github.com/yyhuni/lunafox/releases/tag/v1.1.0")
    expect(releaseLink).toHaveAttribute("target", "_blank")
    expect(releaseLink).toHaveAttribute("rel", "noopener noreferrer")
  })

  it("shows the explicit unavailable state when a development candidate has no notes", () => {
    render(
      <AboutDialogVersionInfo
        t={(key) => key}
        currentVersion="0.0.0-dev"
        candidate={{ ...candidate, releaseVersion: "0.0.0-dev", releaseNotes: undefined }}
        hasUpdate
        checkError={null}
        isChecking={false}
        isCreating={false}
        canStartUpgrade={false}
        operation={{}}
        onCheckUpdate={vi.fn()}
        onStartUpgrade={vi.fn()}
        onRetry={vi.fn()}
      />,
    )

    expect(screen.getByTestId("release-notes-unavailable")).toHaveTextContent("releaseNotesUnavailable")
    expect(screen.getByRole("link", { name: "viewRelease" })).toHaveAttribute("href", "https://github.com/yyhuni/lunafox/releases/tag/v0.0.0-dev")
  })

  it("loads the installed version when the about entry is opened", async () => {
    apiMocks.post.mockImplementation((path: string) => {
      if (path === "/system:checkForUpdates") return Promise.resolve({ data: updateResult })
      throw new Error(`unexpected POST ${path}`)
    })

    const { result, rerender } = renderHookWithProviders(
      ({ enabled }: { enabled: boolean }) => useAboutDialogState({ enabled }),
      { initialProps: { enabled: false } },
    )
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(apiMocks.post).not.toHaveBeenCalled()
    expect(result.current.currentVersion).toBe("-")

    rerender({ enabled: true })
    await waitFor(() => expect(result.current.currentVersion).toBe(updateResult.currentVersion))
    expect(apiMocks.post).toHaveBeenCalledWith("/system:checkForUpdates", {})
  })

  it("does not silently ignore start upgrade when the candidate is ineligible", async () => {
    const diagnostic = { code: "MIGRATION_UNSUPPORTED", reason: "current policy rejects this migration" }
    apiMocks.post.mockImplementation((path: string) => {
      if (path === "/system:checkForUpdates") {
        return Promise.resolve({ data: { ...updateResult, eligible: false, diagnostic } })
      }
      throw new Error(`unexpected POST ${path}`)
    })

    const { result } = renderHookWithProviders(() => useAboutDialogState())
    await act(async () => {
      await result.current.handleCheckUpdate()
    })
    expect(result.current.currentVersion).toBe(updateResult.currentVersion)
    expect(result.current.hasUpdate).toBe(true)
    expect(result.current.checkError).toBe(diagnostic.reason)

    act(() => result.current.handleStartUpgrade())
    expect(result.current.confirmOpen).toBe(false)
  })

  it("opens from the sidebar about trigger and keeps upgrade confirmation stacked above the about dialog", async () => {
    apiMocks.post.mockImplementation((path: string) => {
      if (path === "/system:checkForUpdates") return Promise.resolve({ data: updateResult })
      throw new Error(`unexpected POST ${path}`)
    })
    apiMocks.get.mockRejectedValue({
      isAxiosError: true,
      response: { status: 404, data: { error: { code: "NOT_FOUND", message: "No active upgrade operation." } } },
      message: "not found",
    })

    renderWithProviders(
      <SidebarProvider>
        <AboutDialog>
          <SidebarMenuButton tooltip="about">about</SidebarMenuButton>
        </AboutDialog>
      </SidebarProvider>,
    )

    fireEvent.click(screen.getByRole("button", { name: "about" }))
    expect(await screen.findByText(updateResult.currentVersion)).toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", { name: "checkUpdate" }))
    expect(await screen.findByRole("button", { name: "startUpgrade" })).toBeEnabled()

    fireEvent.click(screen.getByRole("button", { name: "startUpgrade" }))
    expect(await screen.findByRole("alertdialog", { name: "confirmUpgradeTitle" })).toBeInTheDocument()
  })
})
