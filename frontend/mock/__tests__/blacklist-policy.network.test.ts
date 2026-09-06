import { afterEach, beforeEach, describe, expect, it } from "vitest"
import { resetMockBlacklistPolicies } from "@/mock/data/blacklist-policy"
import { resetMockScenario, setMockScenario } from "@/mock/scenarios"
import { readFileSync } from "node:fs"
import path from "node:path"

const API_BASE = "http://localhost/v1"
const handlerSource = readFileSync(path.resolve(process.cwd(), "mock/handlers/index.ts"), "utf8")

async function json(response: Response) {
  return response.json() as Promise<Record<string, unknown>>
}

describe("blacklist policy mock network boundary", () => {
  beforeEach(() => {
    resetMockBlacklistPolicies()
    setMockScenario("happy")
  })

  afterEach(() => {
    resetMockBlacklistPolicies()
    resetMockScenario()
  })

  it("serves canonical global and Target child resources and enforces updateMask", async () => {
    const globalResponse = await fetch(`${API_BASE}/blacklistPolicy`)
    const global = await json(globalResponse)
    const targetResponse = await fetch(`${API_BASE}/targets/1/blacklistPolicy`)
    const target = await json(targetResponse)

    expect(globalResponse.status).toBe(200)
    expect(global).toMatchObject({ name: "blacklistPolicy", etag: expect.any(String) })
    expect(targetResponse.status).toBe(200)
    expect(target).toMatchObject({ name: "targets/1/blacklistPolicy", etag: expect.any(String) })

    const missingMaskResponse = await fetch(`${API_BASE}/blacklistPolicy`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: global.name, patterns: [], etag: global.etag }),
    })
    const missingMask = await json(missingMaskResponse)

    expect(missingMaskResponse.status).toBe(400)
    expect(missingMask).toMatchObject({ error: { code: "INVALID_ARGUMENT" } })
  })

  it("round-trips canonical saves, no-ops, invalid requests, and stale etag conflicts", async () => {
    const currentResponse = await fetch(`${API_BASE}/blacklistPolicy`)
    const current = await json(currentResponse)

    const savedResponse = await fetch(`${API_BASE}/blacklistPolicy?updateMask=patterns`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: current.name,
        patterns: ["EXAMPLE.com.", "192.0.2.17/24", "example.com"],
        etag: current.etag,
      }),
    })
    const saved = await json(savedResponse)
    expect(savedResponse.status).toBe(200)
    expect(saved).toMatchObject({ patterns: ["192.0.2.0/24", "example.com"] })

    const noOpResponse = await fetch(`${API_BASE}/blacklistPolicy?updateMask=patterns`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: saved.name,
        patterns: ["example.com", "192.0.2.0/24"],
        etag: saved.etag,
      }),
    })
    expect(await json(noOpResponse)).toEqual(saved)

    const staleResponse = await fetch(`${API_BASE}/targets/1/blacklistPolicy?updateMask=patterns`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: "targets/1/blacklistPolicy",
        patterns: ["*.example.com"],
        etag: "stale-etag",
      }),
    })
    expect(staleResponse.status).toBe(409)
    expect(await json(staleResponse)).toMatchObject({ error: { code: "ABORTED" } })

    const invalidResponse = await fetch(`${API_BASE}/targets/1/blacklistPolicy?updateMask=patterns`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: "targets/1/blacklistPolicy",
        patterns: ["api.*.example.com"],
        etag: (await json(await fetch(`${API_BASE}/targets/1/blacklistPolicy`))).etag,
      }),
    })
    expect(invalidResponse.status).toBe(400)
    expect(await json(invalidResponse)).toMatchObject({ error: { code: "INVALID_ARGUMENT" } })
  })

  it("does not expose the removed routes or PUT shape", async () => {
    const [legacyGlobal, legacyTarget, putResponse] = await Promise.all([
      fetch(`${API_BASE}/blacklist/rules`),
      fetch(`${API_BASE}/targets/1/blacklist`),
      fetch(`${API_BASE}/blacklistPolicy?updateMask=patterns`, { method: "PUT" }),
    ])

    expect(legacyGlobal.status).toBe(501)
    expect(legacyTarget.status).toBe(501)
    expect(putResponse.status).toBe(501)
  })

  it("keeps the policy handler limited to canonical GET and PATCH branches", () => {
    const globalStart = handlerSource.indexOf('if (method === "GET" && path === "/blacklistPolicy")')
    const globalEnd = handlerSource.indexOf('if (method === "GET" && path === "/databaseHealthReports/current")')
    const targetStart = handlerSource.indexOf("const targetBlacklistMatch")
    const targetEnd = handlerSource.indexOf("const targetDirectoryFilterOptionsMatch")
    const globalHandler = handlerSource.slice(globalStart, globalEnd)
    const targetHandler = handlerSource.slice(targetStart, targetEnd)

    expect(globalHandler).toContain('method === "GET"')
    expect(globalHandler).toContain('method === "PATCH"')
    expect(targetHandler).toContain('method === "GET"')
    expect(targetHandler).toContain('method === "PATCH"')
    expect(globalHandler).not.toContain("PUT")
    expect(targetHandler).not.toContain("PUT")
    expect(handlerSource).not.toContain("/blacklist/rules")
  })
})
