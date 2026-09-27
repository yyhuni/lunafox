import { afterEach, describe, expect, it, vi } from "vitest"

import { resetMockScenario, setMockScenario } from "@/mock/scenarios"
import {
  createMockUpgradeOperation,
  getMockUpdateCheckResult,
  getMockVersionInfo,
  getMockUpgradeOperationFull,
  observeMockUpgradeOperation,
  resetMockUpgradeOperation,
} from "@/mock/data/version"

const requestId = "22222222-2222-4222-8222-222222222222"
const legacyMockUpgradeStateKey = "lunafox.mock.upgrade.state.v1"

const legacyMockUpgradeState = JSON.stringify({
  operation: {
    name: "upgradeOperations/11111111-1111-4111-8111-111111111111",
    operationId: "11111111-1111-4111-8111-111111111111",
    requestId,
    operatorId: 1,
    manifestId: "lunafox-1.2.3",
    manifestDigest: `sha256:${"a".repeat(64)}`,
    currentVersion: "mock-2026.04.26",
    releaseVersion: "mock-2026.05.01",
    compatibilityRange: ">=1.0.0 <2.0.0",
    maintenanceWindowMinutes: 15,
    status: "queued",
    migrationStatus: "not_started",
    migrationType: "none",
    cancelledScanCount: 0,
    cancelledTaskCount: 0,
    agentSummary: { expected: 0, ready: 0, missing: 0, unhealthy: 0 },
    observedDigests: {},
    planSummary: { touchedServices: ["frontend"] },
    logs: [],
    stageTimes: { queued: "2026-09-24T09:49:00Z" },
    createdAt: "2026-09-24T09:49:00Z",
    updatedAt: "2026-09-24T09:49:00Z",
    completedAt: null,
  },
  pollCount: 0,
})

describe("version upgrade mock lifecycle", () => {
  afterEach(() => {
    resetMockUpgradeOperation()
    resetMockScenario()
  })

  it("uses the public repository for release links", () => {
    expect(getMockVersionInfo().githubRepo).toBe("https://github.com/yyhuni/lunafox")
  })

  it("progresses the happy scenario through server-confirmed stages", () => {
    setMockScenario("happy")
    createMockUpgradeOperation({ requestId, manifestId: "lunafox-1.2.3", manifestDigest: `sha256:${"a".repeat(64)}` })

    expect(observeMockUpgradeOperation()?.status).toBe("queued")
    expect(observeMockUpgradeOperation()?.status).toBe("stopping")
    expect(observeMockUpgradeOperation()?.status).toBe("updating")
    expect(observeMockUpgradeOperation()?.status).toBe("migrating")
    expect(observeMockUpgradeOperation()?.status).toBe("restarting")
    expect(observeMockUpgradeOperation()?.status).toBe("verifying")
    expect(observeMockUpgradeOperation()?.status).toBe("succeeded")
  })

  it("keeps edge and stress scenarios distinct from an ordinary retry", () => {
    setMockScenario("edge")
    createMockUpgradeOperation({ requestId, manifestId: "lunafox-1.2.3", manifestDigest: `sha256:${"a".repeat(64)}` })
    observeMockUpgradeOperation()
    observeMockUpgradeOperation()
    observeMockUpgradeOperation()
    expect(observeMockUpgradeOperation()?.status).toBe("needs_attention")

    resetMockUpgradeOperation()
    setMockScenario("stress")
    createMockUpgradeOperation({ requestId, manifestId: "lunafox-1.2.3", manifestDigest: `sha256:${"a".repeat(64)}` })
    for (let index = 0; index < 5; index += 1) observeMockUpgradeOperation()
    expect(observeMockUpgradeOperation()?.status).toBe("failed")
  })

  it("adds bounded safe sub-step events while a long upgrade stage is active", () => {
    setMockScenario("happy")
    createMockUpgradeOperation({ requestId, manifestId: "lunafox-1.2.3", manifestDigest: `sha256:${"a".repeat(64)}` })
    observeMockUpgradeOperation()
    observeMockUpgradeOperation()
    const updating = observeMockUpgradeOperation()

    expect(updating?.status).toBe("updating")
    expect(updating?.logs.map((entry) => entry.messageKey)).toEqual(expect.arrayContaining([
      "pullImagesStarted",
      "servicesUpdateStarted",
      "servicesUpdated",
    ]))
    expect(updating?.logs.some((entry) => /docker compose|\/|secret|token/i.test(entry.message))).toBe(false)
  })

  it("returns host activity only for the enhanced mock FULL projection without heartbeat log rows", () => {
    setMockScenario("happy")
    createMockUpgradeOperation({ requestId, manifestId: "lunafox-1.2.3", manifestDigest: `sha256:${"a".repeat(64)}` })
    observeMockUpgradeOperation()
    observeMockUpgradeOperation()
    expect(observeMockUpgradeOperation()?.status).toBe("updating")

    expect(getMockUpgradeOperationFull()).not.toHaveProperty("hostActivity")
    expect(getMockUpgradeOperationFull(true)).toMatchObject({
      hostActivity: { action: "update_services" },
    })
    expect(getMockUpgradeOperationFull(true)?.logs.some((entry) => /heartbeat/i.test(entry.messageKey))).toBe(false)

    for (let index = 0; index < 4; index += 1) observeMockUpgradeOperation()
    expect(getMockUpgradeOperationFull(true)).toMatchObject({ status: "succeeded", hostActivity: null })
  })

  it("keeps terminal status stable when the active view is polled again", () => {
    setMockScenario("edge")
    createMockUpgradeOperation({ requestId, manifestId: "lunafox-1.2.3", manifestDigest: `sha256:${"a".repeat(64)}` })
    for (let index = 0; index < 4; index += 1) observeMockUpgradeOperation()

    expect(observeMockUpgradeOperation()?.status).toBe("needs_attention")
    expect(observeMockUpgradeOperation()?.status).toBe("needs_attention")
  })

  it("restores the operation after the browser mock module is reloaded", async () => {
    setMockScenario("happy")
    const created = createMockUpgradeOperation({ requestId, manifestId: "lunafox-1.2.3", manifestDigest: `sha256:${"a".repeat(64)}` })
    observeMockUpgradeOperation()
    observeMockUpgradeOperation()

    vi.resetModules()
    const reloadedVersion = await import("../version")
    expect(reloadedVersion.getMockUpgradeOperation()).toMatchObject({
      operationId: created.operationId,
      status: "stopping",
    })
    reloadedVersion.resetMockUpgradeOperation()
  })

  it("discards an obsolete persisted fixture instead of resuming its upgrade", async () => {
    localStorage.setItem(legacyMockUpgradeStateKey, legacyMockUpgradeState)

    vi.resetModules()
    const reloadedVersion = await import("../version")

    expect(reloadedVersion.getMockUpgradeOperation()).toBeNull()
    expect(localStorage.getItem(legacyMockUpgradeStateKey)).toBeNull()
    reloadedVersion.resetMockUpgradeOperation()
  })

  it("starts a new operation after a terminal fixture", () => {
    setMockScenario("happy")
    const first = createMockUpgradeOperation({ requestId, manifestId: "lunafox-1.2.3", manifestDigest: `sha256:${"a".repeat(64)}` })
    for (let index = 0; index < 7; index += 1) observeMockUpgradeOperation()

    const next = createMockUpgradeOperation({ requestId, manifestId: "lunafox-1.2.3", manifestDigest: `sha256:${"a".repeat(64)}` })
    expect(next.status).toBe("queued")
    expect(next.name).toBe(`upgradeOperations/${next.operationId}`)
    expect(next.operationId).not.toBe(first.operationId)
  })

  it("reports the upgraded version after a successful fixture", () => {
    setMockScenario("happy")
    const operation = createMockUpgradeOperation({ requestId, manifestId: "lunafox-1.2.3", manifestDigest: `sha256:${"a".repeat(64)}` })
    for (let index = 0; index < 7; index += 1) observeMockUpgradeOperation()

    expect(getMockUpdateCheckResult()).toMatchObject({
      currentVersion: operation.releaseVersion,
      hasUpdate: false,
      candidate: undefined,
    })
  })
})
