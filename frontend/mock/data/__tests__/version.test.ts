import { afterEach, describe, expect, it, vi } from "vitest"

import { resetMockScenario, setMockScenario } from "@/mock/scenarios"
import {
  createMockUpgradeOperation,
  getMockUpdateCheckResult,
  observeMockUpgradeOperation,
  resetMockUpgradeOperation,
} from "@/mock/data/version"

const requestId = "22222222-2222-4222-8222-222222222222"

describe("version upgrade mock lifecycle", () => {
  afterEach(() => {
    resetMockUpgradeOperation()
    resetMockScenario()
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
