import { getMockScenario } from "../scenarios"
import type { BlacklistPolicy } from "@/types/blacklist-policy.types"

type StoredMockPolicy = {
  patterns: string[]
  revision: number
  updateTime: string
}

export type MockBlacklistPolicyPatch = {
  name?: unknown
  patterns?: unknown
  etag?: unknown
}

export type MockBlacklistPolicyPatchResult =
  | { status: 200; body: BlacklistPolicy }
  | { status: 400 | 409; body: { error: { code: "INVALID_ARGUMENT" | "ABORTED"; message: string } } }

const initialGlobalPolicy: StoredMockPolicy = {
  patterns: [
    "*.education.example",
    "*.government.example",
    "10.0.0.1",
    "172.16.0.0/12",
    "192.168.0.0/16",
  ],
  revision: 1,
  updateTime: "2026-01-10T00:00:00.000Z",
}

const initialTargetPolicies: Record<number, StoredMockPolicy> = {
  1: {
    patterns: ["*.internal.example.com", "192.168.1.0/24"],
    revision: 1,
    updateTime: "2026-01-11T00:00:00.000Z",
  },
  2: {
    patterns: ["*.cdn.example.com", "cdn.example.com"],
    revision: 1,
    updateTime: "2026-01-12T00:00:00.000Z",
  },
}

let mockGlobalPolicy = cloneStoredPolicy(initialGlobalPolicy)
let mockTargetPolicies = cloneTargetPolicies(initialTargetPolicies)

function cloneStoredPolicy(policy: StoredMockPolicy): StoredMockPolicy {
  return {
    patterns: [...policy.patterns],
    revision: policy.revision,
    updateTime: policy.updateTime,
  }
}

function cloneTargetPolicies(source: Record<number, StoredMockPolicy>): Record<number, StoredMockPolicy> {
  return Object.fromEntries(
    Object.entries(source).map(([targetId, policy]) => [Number(targetId), cloneStoredPolicy(policy)])
  )
}

function policyETag(name: string, revision: number): string {
  return `mock-${name.replaceAll("/", "-")}-etag-${revision}`
}

function toPolicy(name: string, policy: StoredMockPolicy): BlacklistPolicy {
  return {
    name,
    patterns: [...policy.patterns],
    etag: policyETag(name, policy.revision),
    updateTime: policy.updateTime,
  }
}

function mockError(
  status: 400 | 409,
  code: "INVALID_ARGUMENT" | "ABORTED",
  message: string
): MockBlacklistPolicyPatchResult {
  return { status, body: { error: { code, message } } }
}

function isIPv4(value: string): boolean {
  const parts = value.split(".")
  return parts.length === 4 && parts.every((part) => {
    if (!/^(0|[1-9]\d{0,2})$/.test(part)) return false
    const parsed = Number(part)
    return parsed >= 0 && parsed <= 255
  })
}

function canonicalizeIPv4(value: string): string {
  return value.split(".").map((part) => String(Number(part))).join(".")
}

function canonicalizeCIDR(value: string): string | null {
  const [rawIP, rawPrefix, ...extra] = value.split("/")
  if (extra.length > 0 || !rawIP || !rawPrefix || !isIPv4(rawIP) || !/^\d{1,2}$/.test(rawPrefix)) {
    return null
  }
  const prefix = Number(rawPrefix)
  if (prefix < 0 || prefix > 32) return null
  const address = rawIP.split(".").reduce((result, part) => (result << 8) | Number(part), 0) >>> 0
  const mask = prefix === 0 ? 0 : (0xffffffff << (32 - prefix)) >>> 0
  const masked = address & mask
  const normalizedIP = [24, 16, 8, 0]
    .map((shift) => String((masked >>> shift) & 0xff))
    .join(".")
  return `${normalizedIP}/${prefix}`
}

function isDomain(value: string): boolean {
  const normalized = value.endsWith(".") ? value.slice(0, -1) : value
  if (!normalized || normalized.length > 253 || normalized.includes("/") || normalized.includes(":")) {
    return false
  }
  const labels = normalized.split(".")
  return labels.length >= 2 && labels.every((label) => (
    /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/i.test(label)
  ))
}

function canonicalizePattern(value: unknown): string | null {
  if (typeof value !== "string") return null
  const trimmed = value.trim()
  if (!trimmed) return null

  const cidr = canonicalizeCIDR(trimmed)
  if (cidr) return cidr
  if (isIPv4(trimmed)) return canonicalizeIPv4(trimmed)
  if (trimmed.startsWith("*.")) {
    const base = trimmed.slice(2)
    return isDomain(base) ? `*.${base.replace(/\.$/, "").toLowerCase()}` : null
  }
  if (trimmed.includes("*")) return null
  return isDomain(trimmed) ? trimmed.replace(/\.$/, "").toLowerCase() : null
}

function canonicalizePatterns(value: unknown): string[] | null {
  if (!Array.isArray(value)) return null
  const canonical = value.map(canonicalizePattern)
  if (canonical.some((pattern) => !pattern)) return null
  return Array.from(new Set(canonical as string[])).sort()
}

function readScenarioPolicy(name: string, stored: StoredMockPolicy): BlacklistPolicy {
  const scenario = getMockScenario()
  if (scenario === "empty") {
    return {
      name,
      patterns: [],
      etag: policyETag(`${name}-empty`, stored.revision),
      updateTime: stored.updateTime,
    }
  }
  if (scenario === "edge" && name.startsWith("targets/")) {
    return {
      name,
      patterns: [],
      etag: policyETag(`${name}-edge`, stored.revision),
      updateTime: stored.updateTime,
    }
  }
  return toPolicy(name, stored)
}

function getOrCreateTargetPolicy(targetId: number): StoredMockPolicy {
  const existing = mockTargetPolicies[targetId]
  if (existing) return existing
  const created: StoredMockPolicy = {
    patterns: [],
    revision: 1,
    updateTime: "2026-01-01T00:00:00.000Z",
  }
  mockTargetPolicies[targetId] = created
  return created
}

function patchPolicy(
  name: string,
  stored: StoredMockPolicy,
  body: MockBlacklistPolicyPatch
): MockBlacklistPolicyPatchResult {
  if (body.name !== name || typeof body.etag !== "string" || !body.etag) {
    return mockError(400, "INVALID_ARGUMENT", "name and etag are required")
  }
  const current = readScenarioPolicy(name, stored)
  if (body.etag !== current.etag) {
    return mockError(409, "ABORTED", "blacklist policy was modified; reload before saving")
  }
  const patterns = canonicalizePatterns(body.patterns)
  if (!patterns) {
    return mockError(400, "INVALID_ARGUMENT", "patterns must contain supported blacklist rules")
  }
  if (patterns.join("\u0000") === current.patterns.join("\u0000")) {
    return { status: 200, body: current }
  }

  stored.patterns = patterns
  stored.revision += 1
  stored.updateTime = new Date(Date.UTC(2026, 0, 1, 0, 0, stored.revision)).toISOString()
  return { status: 200, body: toPolicy(name, stored) }
}

export function getMockGlobalBlacklistPolicy(): BlacklistPolicy {
  return readScenarioPolicy("blacklistPolicy", mockGlobalPolicy)
}

export function patchMockGlobalBlacklistPolicy(
  body: MockBlacklistPolicyPatch
): MockBlacklistPolicyPatchResult {
  return patchPolicy("blacklistPolicy", mockGlobalPolicy, body)
}

export function getMockTargetBlacklistPolicy(targetId: number): BlacklistPolicy {
  const name = `targets/${targetId}/blacklistPolicy`
  return readScenarioPolicy(name, getOrCreateTargetPolicy(targetId))
}

export function patchMockTargetBlacklistPolicy(
  targetId: number,
  body: MockBlacklistPolicyPatch
): MockBlacklistPolicyPatchResult {
  const name = `targets/${targetId}/blacklistPolicy`
  return patchPolicy(name, getOrCreateTargetPolicy(targetId), body)
}

export function resetMockBlacklistPolicies(): void {
  mockGlobalPolicy = cloneStoredPolicy(initialGlobalPolicy)
  mockTargetPolicies = cloneTargetPolicies(initialTargetPolicies)
}
