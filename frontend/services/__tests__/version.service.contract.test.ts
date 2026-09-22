import { beforeEach, describe, expect, it, vi } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "services/version.service.ts"), "utf8")

const apiMocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))
vi.mock("@/lib/api-client", () => ({ api: apiMocks }))

import { api } from "@/lib/api-client"
import { VersionService } from "@/services/version.service"

const digest = `sha256:${"a".repeat(64)}`
const releaseNotesBody = "## English\n\n- Test release notes.\n\n## 简体中文\n\n- 测试发布说明。\n"
const releaseNotesSha256 = "sha256:4406112ce062dd05feacce5f519b8cb7250fd01c7237c43e0f7335da912e8188"
const operation = {
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
}
const fullOperation = {
  ...operation,
  executionMode: "frontend_only",
  workDisposition: "not_required",
  planSummary: { touchedServices: ["frontend"] },
  confirmedDeploymentVersion: "1.1.0",
} as const
const fullPlanServices = ["agent", "bootstrap", "engine", "engine_package", "engine_runtime", "frontend", "migration", "nginx", "server"]

describe("version.service contract", () => {
  beforeEach(() => vi.clearAllMocks())
  it("preserves current source markers", () => {
    expect(source).toContain("from '@/lib/api-client'")
  })

  it("checks the server-owned manifest without accepting client target data", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: {
      currentVersion: "1.0.0", hasUpdate: true, eligible: true,
      candidate: {
        name: "releaseManifests/release-1.1.0", manifestId: "release-1.1.0", manifestDigest: digest,
        releaseVersion: "1.1.0", deploymentMode: "single-node-compose", compatibilityRange: ">=1.0.0 <2.0.0",
        maintenanceWindowMinutes: 15, requiresAdminConfirmation: true,
        databaseMigration: { hasDatabaseMigration: false, migrationType: "none", policyVersion: 1 },
        runtimeImageDigests: { server: digest, frontend: digest, nginx: digest }, engineDigests: [digest],
        releaseNotes: { body: releaseNotesBody, sha256: releaseNotesSha256 },
      },
    } } as never)
    const result = await VersionService.checkForUpdates()
    expect(api.post).toHaveBeenCalledWith("/system:checkForUpdates", {})
    expect(result.candidate?.manifestDigest).toBe(digest)
    expect(result.candidate?.releaseNotes?.sha256).toBe(releaseNotesSha256)
  })

  it("rejects tampered release notes and permits the explicit development omission", async () => {
    vi.mocked(api.post).mockResolvedValueOnce({ data: {
      currentVersion: "0.0.0-dev", hasUpdate: false, eligible: true,
      candidate: {
        name: "releaseManifests/lunafox-0.0.0-dev", manifestId: "lunafox-0.0.0-dev", manifestDigest: digest,
        releaseVersion: "0.0.0-dev", deploymentMode: "single-node-compose", compatibilityRange: ">=0.0.0 <1.0.0",
        maintenanceWindowMinutes: 15, requiresAdminConfirmation: true,
        databaseMigration: { hasDatabaseMigration: false, migrationType: "none", policyVersion: 1 },
        runtimeImageDigests: { server: digest }, engineDigests: [],
      },
    } } as never)
    await expect(VersionService.checkForUpdates()).resolves.toMatchObject({ candidate: { releaseVersion: "0.0.0-dev" } })

    vi.mocked(api.post).mockResolvedValueOnce({ data: {
      currentVersion: "1.0.0", hasUpdate: true, eligible: true,
      candidate: {
        name: "releaseManifests/release-1.1.0", manifestId: "release-1.1.0", manifestDigest: digest,
        releaseVersion: "1.1.0", deploymentMode: "single-node-compose", compatibilityRange: ">=1.0.0 <2.0.0",
        maintenanceWindowMinutes: 15, requiresAdminConfirmation: true,
        databaseMigration: { hasDatabaseMigration: false, migrationType: "none", policyVersion: 1 },
        runtimeImageDigests: { server: digest }, engineDigests: [],
        releaseNotes: { body: releaseNotesBody.replace("Test", "Tampered"), sha256: releaseNotesSha256 },
      },
    } } as never)
    await expect(VersionService.checkForUpdates()).rejects.toThrow("candidate.releaseNotes.sha256")
  })

  it("requires confirmation before creating an operation and sends only immutable identity", async () => {
    await expect(VersionService.createUpgradeOperation({
      requestId: operation.requestId, manifestId: operation.manifestId, manifestDigest: digest, confirmed: false,
    })).rejects.toThrow("explicit confirmation")
    expect(api.post).not.toHaveBeenCalled()
    vi.mocked(api.post).mockResolvedValue({ data: operation } as never)
    await VersionService.createUpgradeOperation({ requestId: operation.requestId, manifestId: operation.manifestId, manifestDigest: digest, confirmed: true })
    expect(api.post).toHaveBeenCalledWith("/upgradeOperations", {
      requestId: operation.requestId, manifestId: operation.manifestId, manifestDigest: digest, confirmed: true,
    })
    expect(vi.mocked(api.post).mock.calls[0]?.[1]).not.toHaveProperty("imageRef")
  })

  it("uses the durable operation resource for reconnect and retry", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: operation } as never)
    vi.mocked(api.post).mockResolvedValue({ data: { ...operation, status: "restarting" } } as never)
    await VersionService.getUpgradeOperation(operation.operationId)
    await VersionService.retryUpgradeOperation(operation.operationId)
    expect(api.get).toHaveBeenCalledWith(`/upgradeOperations/${operation.operationId}`)
    expect(api.post).toHaveBeenCalledWith(`/upgradeOperations/${operation.operationId}:retry`, { confirmed: true })
  })

  it("requests and strictly decodes the explicit FULL projection without changing BASIC", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: fullOperation } as never)

    await expect(VersionService.getUpgradeOperationFull(operation.operationId)).resolves.toMatchObject({
      executionMode: "frontend_only",
      workDisposition: "not_required",
      planSummary: { touchedServices: ["frontend"] },
      confirmedDeploymentVersion: "1.1.0",
    })
    expect(api.get).toHaveBeenCalledWith(`/upgradeOperations/${operation.operationId}`, { params: { view: "FULL" } })

    vi.mocked(api.get).mockResolvedValueOnce({ data: operation } as never)
    await expect(VersionService.getUpgradeOperationFull(operation.operationId)).rejects.toThrow("upgradeOperation.executionMode")

    vi.mocked(api.get).mockResolvedValueOnce({ data: {
      ...fullOperation,
      planSummary: { touchedServices: ["frontend"], unexpected: true },
    } } as never)
    await expect(VersionService.getUpgradeOperationFull(operation.operationId)).rejects.toThrow("upgradeOperation.planSummary.unexpected")
  })

  it("permits empty scope facts only for explicit legacy full operations", async () => {
    const legacy = {
      ...operation,
      executionMode: "full",
      workDisposition: "legacy_unknown",
      planSummary: { touchedServices: [] },
      confirmedDeploymentVersion: "",
    }
    vi.mocked(api.get).mockResolvedValueOnce({ data: legacy } as never)
    await expect(VersionService.getUpgradeOperationFull(operation.operationId)).resolves.toMatchObject({
      executionMode: "full",
      workDisposition: "legacy_unknown",
      planSummary: { touchedServices: [] },
      confirmedDeploymentVersion: "",
    })

    vi.mocked(api.get).mockResolvedValueOnce({ data: { ...legacy, workDisposition: "cancelled" } } as never)
    await expect(VersionService.getUpgradeOperationFull(operation.operationId)).rejects.toThrow("upgradeOperation.planSummary.touchedServices")
  })

  it("requires a complete scoped service set for non-legacy full operations", async () => {
    const full = {
      ...operation,
      executionMode: "full",
      workDisposition: "cancelled",
      planSummary: { touchedServices: fullPlanServices },
      confirmedDeploymentVersion: "",
    }
    vi.mocked(api.get).mockResolvedValueOnce({ data: full } as never)
    await expect(VersionService.getUpgradeOperationFull(operation.operationId)).resolves.toMatchObject({
      executionMode: "full",
      workDisposition: "cancelled",
      planSummary: { touchedServices: fullPlanServices },
      confirmedDeploymentVersion: "",
    })

    vi.mocked(api.get).mockResolvedValueOnce({ data: {
      ...full,
      planSummary: { touchedServices: ["agent", "bootstrap", "frontend", "nginx", "server"] },
    } } as never)
    await expect(VersionService.getUpgradeOperationFull(operation.operationId)).rejects.toThrow("upgradeOperation.planSummary.touchedServices")

    vi.mocked(api.get).mockResolvedValueOnce({ data: { ...full, workDisposition: "not_required" } } as never)
    await expect(VersionService.getUpgradeOperationFull(operation.operationId)).rejects.toThrow("upgradeOperation.workDisposition")
  })

  it("maps an empty active-operation view to null and sends an explicit stop", async () => {
    const notFound = Object.assign(new Error("not found"), {
      isAxiosError: true,
      response: { status: 404, data: { error: { code: "NOT_FOUND", message: "No active upgrade operation." } } },
    })
    vi.mocked(api.get).mockRejectedValueOnce(notFound)
    await expect(VersionService.getActiveUpgradeOperation()).resolves.toBeNull()
    expect(api.get).toHaveBeenCalledWith("/upgradeOperations:active")

    vi.mocked(api.post).mockResolvedValueOnce({ data: operation } as never)
    await VersionService.stopUpgradeOperation(operation.operationId)
    expect(api.post).toHaveBeenCalledWith(`/upgradeOperations/${operation.operationId}:stop`, { confirmed: true })
  })

  it("rejects oversized or unsafe upgrade log projections at the service boundary", async () => {
    const oversized = Array.from({ length: 33 }, (_, index) => ({
      timestamp: `2026-09-13T12:${String(index).padStart(2, "0")}:00Z`,
      level: "info",
      stage: "queued",
      messageKey: "requestAccepted",
      message: "Upgrade request accepted",
    }))
    vi.mocked(api.get).mockResolvedValue({ data: { ...operation, logs: oversized } } as never)
    await expect(VersionService.getUpgradeOperation(operation.operationId)).rejects.toThrow("upgradeOperation.logs")

    vi.mocked(api.get).mockResolvedValue({ data: {
      ...operation,
      logs: [{
        timestamp: "2026-09-13T12:00:00Z",
        level: "error",
        stage: "failed",
        messageKey: "diagnostic",
        message: "executor failed at /srv/lunafox/.env TOKEN=secret",
      }],
    } } as never)
    await expect(VersionService.getUpgradeOperation(operation.operationId)).rejects.toThrow("upgradeOperation.logs[0].message")

    vi.mocked(api.get).mockResolvedValue({ data: {
      ...operation,
      logs: [{
        timestamp: "2026-09-13T12:00:00Z",
        level: "info",
        stage: "updating",
        messageKey: "pullImagesStarted",
        message: "Pulling release images",
        metadata: { secret: "not-allowed" },
      }],
    } } as never)
    await expect(VersionService.getUpgradeOperation(operation.operationId)).rejects.toThrow("upgradeOperation.logs[0].metadata.secret")
  })

  it("accepts bounded progress entries from the existing logs projection", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: {
      ...operation,
      logs: [{
        timestamp: "2026-09-13T12:02:00Z",
        level: "info",
        stage: "updating",
        messageKey: "pullImagesStarted",
        message: "Pulling release images",
        metadata: { scope: "release" },
      }],
    } } as never)

    await expect(VersionService.getUpgradeOperation(operation.operationId)).resolves.toMatchObject({
      logs: [{ messageKey: "pullImagesStarted", message: "Pulling release images", metadata: { scope: "release" } }],
    })
  })
})
