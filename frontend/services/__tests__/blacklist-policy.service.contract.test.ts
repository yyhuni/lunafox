import { beforeEach, describe, expect, it, vi } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const apiMocks = vi.hoisted(() => ({
  get: vi.fn(),
  patch: vi.fn(),
  put: vi.fn(),
}))

vi.mock("@/lib/api-client", () => ({
  api: apiMocks,
}))

import { api } from "@/lib/api-client"
import {
  getGlobalBlacklistPolicy,
  getTargetBlacklistPolicy,
  updateGlobalBlacklistPolicy,
  updateTargetBlacklistPolicy,
} from "@/services/blacklist-policy.service"

const source = readFileSync(path.resolve(process.cwd(), "services/blacklist-policy.service.ts"), "utf8")

const globalPolicy = {
  name: "blacklistPolicy",
  patterns: ["*.example.com", "192.0.2.0/24"],
  etag: "global-etag",
  updateTime: "2026-08-05T00:00:00Z",
}

const targetPolicy = {
  name: "targets/42/blacklistPolicy",
  patterns: ["*.internal.example.com"],
  etag: "target-etag",
  updateTime: "2026-08-05T00:00:00Z",
}

describe("blacklist-policy.service contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("reads global and Target child singleton resources through canonical paths", async () => {
    vi.mocked(api.get)
      .mockResolvedValueOnce({ data: globalPolicy } as never)
      .mockResolvedValueOnce({ data: targetPolicy } as never)

    const [global, target] = await Promise.all([
      getGlobalBlacklistPolicy(),
      getTargetBlacklistPolicy(42),
    ])

    expect(api.get).toHaveBeenNthCalledWith(1, "/blacklistPolicy")
    expect(api.get).toHaveBeenNthCalledWith(2, "/targets/42/blacklistPolicy")
    expect(global).toEqual(globalPolicy)
    expect(target).toEqual(targetPolicy)
    expect(global.patterns).not.toBe(globalPolicy.patterns)
    expect(target.patterns).not.toBe(targetPolicy.patterns)
  })

  it("PATCHes the global singleton with only name, patterns, etag, and updateMask=patterns", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: globalPolicy } as never)

    await updateGlobalBlacklistPolicy({
      patterns: ["*.example.com", "192.0.2.0/24"],
      etag: "global-etag",
    })

    expect(api.patch).toHaveBeenCalledWith(
      "/blacklistPolicy",
      {
        name: "blacklistPolicy",
        patterns: ["*.example.com", "192.0.2.0/24"],
        etag: "global-etag",
      },
      { params: { updateMask: "patterns" } }
    )
    expect(Object.keys(vi.mocked(api.patch).mock.calls[0]?.[1] ?? {}).sort()).toEqual([
      "etag",
      "name",
      "patterns",
    ])
  })

  it("PATCHes only the Target child resource with the Target-local etag", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: targetPolicy } as never)

    await updateTargetBlacklistPolicy(42, {
      patterns: ["*.internal.example.com"],
      etag: "target-etag",
    })

    expect(api.patch).toHaveBeenCalledWith(
      "/targets/42/blacklistPolicy",
      {
        name: "targets/42/blacklistPolicy",
        patterns: ["*.internal.example.com"],
        etag: "target-etag",
      },
      { params: { updateMask: "patterns" } }
    )
  })

  it("does not retain the placeholder transport contract or a no-etag fallback", () => {
    expect(source).toContain('"/blacklistPolicy"')
    expect(source).toContain("targetBlacklistPolicyName")
    expect(source).toContain('updateMask: "patterns"')
    expect(source).not.toContain("/blacklist/rules")
    expect(source).not.toContain("/targets/${targetId}/blacklist")
    expect(source).not.toContain("api.put")
    expect(api.put).not.toHaveBeenCalled()
  })
})
