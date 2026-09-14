import { beforeEach, describe, expect, it, vi } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "services/version.service.ts"), "utf8")

const apiMocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))
vi.mock("@/lib/api-client", () => ({ api: apiMocks }))

import { api } from "@/lib/api-client"
import { VersionService } from "@/services/version.service"

const digest = `sha256:${"a".repeat(64)}`
const operation = {
  name: "upgradeOperations/11111111-1111-4111-8111-111111111111",
  operationId: "11111111-1111-4111-8111-111111111111",
  requestId: "22222222-2222-4222-8222-222222222222",
  operatorId: 7,
  manifestId: "release-1.1.0",
  manifestDigest: digest,
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
  stageTimes: { queued: "2026-09-13T12:00:00Z" },
  createdAt: "2026-09-13T12:00:00Z",
  updatedAt: "2026-09-13T12:00:00Z",
}

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
      },
    } } as never)
    const result = await VersionService.checkForUpdates()
    expect(api.post).toHaveBeenCalledWith("/system:checkForUpdates", {})
    expect(result.candidate?.manifestDigest).toBe(digest)
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
})
