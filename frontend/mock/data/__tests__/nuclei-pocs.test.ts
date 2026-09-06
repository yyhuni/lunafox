import { afterEach, describe, expect, it } from "vitest"

import {
  createMockNucleiPocSync,
  getMockNucleiPocFilterOptions,
  getMockNucleiPocs,
  getMockNucleiPocSyncTask,
  resetMockNucleiPocs,
  setMockNucleiPocActivation,
  updateMockNucleiPocEnabled,
} from "@/mock/data/nuclei-pocs"

const baseRequest = {
  requestId: "00000000-0000-4000-8000-000000000101",
  sourceType: "git" as const,
  repoUrl: "https://github.com/projectdiscovery/nuclei-templates.git",
}

describe("nuclei POC network mock", () => {
  afterEach(() => resetMockNucleiPocs())

  it("advances task phases on each poll and preserves persisted enablement", () => {
    const created = createMockNucleiPocSync(baseRequest)
    expect(created.task?.state).toBe("VALIDATING_SOURCE")
    expect(getMockNucleiPocSyncTask(created.task!.name)?.state).toBe("CLONING")
    expect(getMockNucleiPocSyncTask(created.task!.name)?.state).toBe("SCANNING_FILES")
    for (let index = 0; index < 4; index += 1) getMockNucleiPocSyncTask(created.task!.name)
    expect(getMockNucleiPocSyncTask(created.task!.name)?.state).toBe("SUCCEEDED")

    const updated = updateMockNucleiPocEnabled("nucleiPocs/http-missing-security-headers", false)
    expect(updated?.isEnabled).toBe(false)
    expect(getMockNucleiPocs({}).results.find((item) => item.templateId === "http-missing-security-headers")?.isEnabled).toBe(false)
  })

  it("models active conflicts, replay conflicts, expiry, and approved filters", () => {
    const created = createMockNucleiPocSync(baseRequest)
    expect(createMockNucleiPocSync({ ...baseRequest, requestId: "00000000-0000-4000-8000-000000000102" }).conflictTaskName).toBe(created.task?.name)
    expect(createMockNucleiPocSync(baseRequest).task?.name).toBe(created.task?.name)

    expect(() => createMockNucleiPocSync({ ...baseRequest, repoUrl: "https://example.com/other.git" })).toThrow()
    const high = getMockNucleiPocs({ filter: 'severity=="high"' })
    expect(high.results.every((item) => item.severity === "high")).toBe(true)
    expect(() => getMockNucleiPocs({ filter: 'cve=="CVE-2024-3400"' })).toThrow()
  })

  it("aggregates complete-catalog tag filter options with stable counts and ordering", () => {
    expect(getMockNucleiPocFilterOptions("tags").results).toEqual([
      { value: "actuator", label: "actuator", count: 1 },
      { value: "cve", label: "cve", count: 1 },
      { value: "exposure", label: "exposure", count: 1 },
      { value: "globalprotect", label: "globalprotect", count: 1 },
      { value: "headers", label: "headers", count: 1 },
      { value: "http", label: "http", count: 1 },
      { value: "misconfig", label: "misconfig", count: 1 },
      { value: "paloalto", label: "paloalto", count: 1 },
      { value: "rce", label: "rce", count: 1 },
      { value: "spring", label: "spring", count: 1 },
    ])
    expect(() => getMockNucleiPocFilterOptions("severity" as never)).toThrow()
  })

  it("applies activation to the complete mock catalog and counts actual changes", () => {
    expect(setMockNucleiPocActivation(false)).toEqual({ enabled: false, affectedCount: 2 })
    expect(
      getMockNucleiPocs({}).results.find(
        (item) => item.templateId === "CVE-2024-3400",
      )?.isEnabled,
    ).toBe(false)
    expect(setMockNucleiPocActivation(false)).toEqual({ enabled: false, affectedCount: 0 })
    expect(setMockNucleiPocActivation(true)).toEqual({ enabled: true, affectedCount: 3 })
  })

  it("releases an expired task slot without allowing request replay", () => {
    const expiredRequest = { ...baseRequest, requestId: "00000000-0000-4000-8000-000000000103", repoUrl: "https://example.com/expired.git" }
    const created = createMockNucleiPocSync(expiredRequest)
    expect(getMockNucleiPocSyncTask(created.task!.name)).toBeNull()
    expect(() => createMockNucleiPocSync(expiredRequest)).toThrow()
    expect(createMockNucleiPocSync({ ...baseRequest, requestId: "00000000-0000-4000-8000-000000000104" }).task).toBeTruthy()
  })
})
