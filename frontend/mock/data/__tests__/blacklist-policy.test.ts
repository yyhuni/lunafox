import { afterEach, beforeEach, describe, expect, it } from "vitest"
import {
  getMockGlobalBlacklistPolicy,
  getMockTargetBlacklistPolicy,
  patchMockGlobalBlacklistPolicy,
  patchMockTargetBlacklistPolicy,
  resetMockBlacklistPolicies,
} from "@/mock/data/blacklist-policy"
import { resetMockScenario, setMockScenario } from "@/mock/scenarios"

describe("blacklist policy mock data", () => {
  beforeEach(() => {
    resetMockBlacklistPolicies()
    setMockScenario("happy")
  })

  afterEach(() => {
    resetMockBlacklistPolicies()
    resetMockScenario()
  })

  it("provides separate global and Target-local policies, including empty and edge scenarios", () => {
    expect(getMockGlobalBlacklistPolicy()).toMatchObject({ name: "blacklistPolicy" })
    expect(getMockTargetBlacklistPolicy(1)).toMatchObject({ name: "targets/1/blacklistPolicy" })

    setMockScenario("empty")
    expect(getMockGlobalBlacklistPolicy().patterns).toEqual([])
    expect(getMockTargetBlacklistPolicy(1).patterns).toEqual([])

    setMockScenario("edge")
    expect(getMockGlobalBlacklistPolicy().patterns).not.toEqual([])
    expect(getMockTargetBlacklistPolicy(1).patterns).toEqual([])
  })

  it("returns a canonical global policy that can be resubmitted as a no-op", () => {
    const current = getMockGlobalBlacklistPolicy()
    const result = patchMockGlobalBlacklistPolicy({
      name: current.name,
      patterns: current.patterns,
      etag: current.etag,
    })

    expect(result).toEqual({ status: 200, body: current })
  })

  it("canonicalizes saves and preserves etag/updateTime for a semantic no-op", () => {
    const before = getMockGlobalBlacklistPolicy()
    const saved = patchMockGlobalBlacklistPolicy({
      name: before.name,
      patterns: ["EXAMPLE.com.", "192.0.2.17/24", "example.com"],
      etag: before.etag,
    })

    expect(saved.status).toBe(200)
    if (saved.status !== 200) throw new Error("expected a successful mock policy save")
    expect(saved.body.patterns).toEqual(["192.0.2.0/24", "example.com"])
    expect(saved.body.etag).not.toBe(before.etag)

    const noOp = patchMockGlobalBlacklistPolicy({
      name: saved.body.name,
      patterns: [...saved.body.patterns].reverse(),
      etag: saved.body.etag,
    })

    expect(noOp.status).toBe(200)
    if (noOp.status !== 200) throw new Error("expected a successful mock no-op")
    expect(noOp.body).toEqual(saved.body)
  })

  it("rejects stale etags and invalid patterns without changing Target-local state", () => {
    const before = getMockTargetBlacklistPolicy(7)
    const stale = patchMockTargetBlacklistPolicy(7, {
      name: before.name,
      patterns: ["*.example.com"],
      etag: "stale-etag",
    })
    expect(stale).toEqual({
      status: 409,
      body: { error: { code: "ABORTED", message: "blacklist policy was modified; reload before saving" } },
    })

    const invalid = patchMockTargetBlacklistPolicy(7, {
      name: before.name,
      patterns: ["api.*.example.com"],
      etag: before.etag,
    })
    expect(invalid.status).toBe(400)
    if (invalid.status === 200) throw new Error("expected an invalid mock policy save")
    expect(invalid.body.error.code).toBe("INVALID_ARGUMENT")
    expect(getMockTargetBlacklistPolicy(7)).toEqual(before)
  })
})
