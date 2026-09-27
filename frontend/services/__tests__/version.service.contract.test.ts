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
  agentDiagnostics: [],
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

  it("accepts missing release notes projections and rejects tampered release notes", async () => {
    vi.mocked(api.post).mockResolvedValueOnce({ data: {
      currentVersion: "0.0.1-alpha.164", hasUpdate: true, eligible: true,
      candidate: {
        name: "releaseManifests/lunafox-0.0.1-alpha.183", manifestId: "lunafox-0.0.1-alpha.183", manifestDigest: digest,
        releaseVersion: "0.0.1-alpha.183", deploymentMode: "single-node-compose", compatibilityRange: ">=0.0.1-alpha.164 <2.0.0",
        maintenanceWindowMinutes: 15, requiresAdminConfirmation: true,
        databaseMigration: { hasDatabaseMigration: false, migrationType: "none", policyVersion: 1 },
        runtimeImageDigests: { server: digest }, engineDigests: [],
      },
    } } as never)
    const bridgeResult = await VersionService.checkForUpdates()
    expect(bridgeResult.candidate).toMatchObject({ releaseVersion: "0.0.1-alpha.183" })
    expect(bridgeResult.candidate?.releaseNotes).toBeUndefined()

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

    const legacyFull = await VersionService.getUpgradeOperationFull(operation.operationId)
    expect(legacyFull).toMatchObject({
      executionMode: "frontend_only",
      workDisposition: "not_required",
      planSummary: { touchedServices: ["frontend"] },
      confirmedDeploymentVersion: "1.1.0",
    })
    expect(legacyFull.hostActivity).toBeUndefined()
    expect(api.get).toHaveBeenCalledWith(`/upgradeOperations/${operation.operationId}`, {
      params: { view: "FULL", includeHostActivity: "true" },
    })

    const diagnosticOperation = {
      ...fullOperation,
      executionMode: "full" as const,
      workDisposition: "cancelled" as const,
      planSummary: { touchedServices: fullPlanServices },
      agentDiagnostics: [{
        agentId: 9,
        name: "agents/9",
        displayNameSnapshot: "edge-9",
        reasonCode: "heartbeat_missing_or_stale",
        detail: "Agent heartbeat is missing or stale",
        source: "server_observation" as const,
      }],
    }
    vi.mocked(api.get).mockResolvedValueOnce({ data: diagnosticOperation } as never)
    await expect(VersionService.getUpgradeOperationFull(operation.operationId)).resolves.toMatchObject({
      agentDiagnostics: [{ agentId: 9, name: "agents/9", displayNameSnapshot: "edge-9", source: "server_observation" }],
    })

    vi.mocked(api.get).mockResolvedValueOnce({ data: operation } as never)
    await expect(VersionService.getUpgradeOperationFull(operation.operationId)).rejects.toThrow("upgradeOperation.executionMode")

    vi.mocked(api.get).mockResolvedValueOnce({ data: {
      ...fullOperation,
      planSummary: { touchedServices: ["frontend"], unexpected: true },
    } } as never)
    await expect(VersionService.getUpgradeOperationFull(operation.operationId)).rejects.toThrow("upgradeOperation.planSummary.unexpected")
  })

  it("accepts only a valid enhanced host activity while preserving older FULL responses", async () => {
    const active = {
      ...fullOperation,
      status: "updating" as const,
      stageTimes: { queued: "2026-09-13T12:00:00Z", updating: "2026-09-13T12:01:00Z" },
      updatedAt: "2026-09-13T12:05:00Z",
      hostActivity: {
        action: "pull_images",
        startedAt: "2026-09-13T12:01:00Z",
        lastHeartbeatAt: "2026-09-13T12:04:00Z",
      },
    }
    vi.mocked(api.get).mockResolvedValueOnce({ data: active } as never)
    await expect(VersionService.getUpgradeOperationFull(operation.operationId)).resolves.toMatchObject({
      hostActivity: active.hostActivity,
    })

    const invalidCases: Array<[string, Record<string, unknown>]> = [
      ["upgradeOperation.hostActivity.action", { action: "unknown_action" }],
      ["upgradeOperation.hostActivity.action", { action: "database_migration" }],
      ["upgradeOperation.hostActivity", { startedAt: "2026-09-13T12:05:00Z", lastHeartbeatAt: "2026-09-13T12:04:00Z" }],
      ["upgradeOperation.hostActivity.unexpected", { unexpected: true }],
    ]
    for (const [path, change] of invalidCases) {
      vi.mocked(api.get).mockResolvedValueOnce({
        data: { ...active, hostActivity: { ...active.hostActivity, ...change } },
      } as never)
      await expect(VersionService.getUpgradeOperationFull(operation.operationId)).rejects.toThrow(path)
    }

    vi.mocked(api.get).mockResolvedValueOnce({
      data: { ...active, status: "succeeded", completedAt: "2026-09-13T12:05:00Z" },
    } as never)
    await expect(VersionService.getUpgradeOperationFull(operation.operationId)).rejects.toThrow("upgradeOperation.hostActivity")

    vi.mocked(api.get).mockResolvedValueOnce({ data: { ...active, hostActivity: null } } as never)
    await expect(VersionService.getUpgradeOperationFull(operation.operationId)).resolves.toMatchObject({ hostActivity: null })

    const oldServer = { ...active } as Record<string, unknown>
    delete oldServer.hostActivity
    vi.mocked(api.get).mockResolvedValueOnce({ data: oldServer } as never)
    const oldServerResponse = await VersionService.getUpgradeOperationFull(operation.operationId)
    expect(oldServerResponse.hostActivity).toBeUndefined()

    vi.mocked(api.get).mockResolvedValueOnce({ data: active } as never)
    await expect(VersionService.getActiveUpgradeOperationFull()).resolves.toMatchObject({ hostActivity: active.hostActivity })
    expect(api.get).toHaveBeenLastCalledWith("/upgradeOperations:active", {
      params: { view: "FULL", includeHostActivity: "true" },
    })

    vi.mocked(api.get).mockResolvedValueOnce({ data: { ...operation, hostActivity: null } } as never)
    await expect(VersionService.getUpgradeOperation(operation.operationId)).rejects.toThrow("upgradeOperation.hostActivity")
  })

  it("rejects malformed Agent diagnostic fields instead of inferring codes from text", async () => {
    const base = {
      ...fullOperation,
      executionMode: "full" as const,
      workDisposition: "cancelled" as const,
      planSummary: { touchedServices: fullPlanServices },
      agentDiagnostics: [{
        agentId: 9,
        name: "agents/9",
        reasonCode: "legacy_unknown",
        detail: "No reliable diagnostic was recorded",
        source: "historical" as const,
      }],
    }
    const invalidCases: Array<[string, Record<string, unknown>]> = [
      ["source", { source: "free_text" }],
      ["reasonCode", { reasonCode: "new_reason" }],
      ["detail", { detail: "Authorization token leaked" }],
      ["name", { name: "agents/10" }],
    ]
    for (const [field, change] of invalidCases) {
      vi.mocked(api.get).mockResolvedValueOnce({
        data: { ...base, agentDiagnostics: [{ ...base.agentDiagnostics[0], ...change }] },
      } as never)
      await expect(VersionService.getUpgradeOperationFull(operation.operationId)).rejects.toThrow(`upgradeOperation.agentDiagnostics[0].${field}`)
    }

    vi.mocked(api.get).mockResolvedValueOnce({ data: { ...base, agentDiagnostics: [] } } as never)
    await expect(VersionService.getUpgradeOperationFull(operation.operationId)).resolves.toMatchObject({ agentDiagnostics: [] })

    const missingDiagnostics = { ...base } as Record<string, unknown>
    delete missingDiagnostics.agentDiagnostics
    vi.mocked(api.get).mockResolvedValueOnce({ data: missingDiagnostics } as never)
    await expect(VersionService.getUpgradeOperationFull(operation.operationId)).rejects.toThrow("upgradeOperation.agentDiagnostics")

    vi.mocked(api.get).mockResolvedValueOnce({ data: { ...operation, agentDiagnostics: [] } } as never)
    await expect(VersionService.getUpgradeOperation(operation.operationId)).rejects.toThrow("upgradeOperation.agentDiagnostics")
  })

  it("permits empty scope facts only for explicit legacy full operations", async () => {
    const legacy = {
      ...operation,
      executionMode: "full",
      workDisposition: "legacy_unknown",
      planSummary: { touchedServices: [] },
      confirmedDeploymentVersion: "",
      agentDiagnostics: [],
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
      agentDiagnostics: [],
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
