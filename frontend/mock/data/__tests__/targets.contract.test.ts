import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"
import {
  bulkDeleteMockTargets,
  deleteMockTarget,
  getMockTargetById,
  getMockTargets,
  mockTargets,
} from "../targets"
import { getMockWebsites } from "../websites"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/targets.ts"), "utf8")

describe("targets contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function getMockTargets")
    expect(source).toContain("from '@/types/target.types'")
  })

  it("keeps the first mock page rich enough to exercise table edge cases", () => {
    const response = getMockTargets({ page: 1, pageSize: 10 })
    const firstPage = response.results

    expect(response.nextPageToken).toBe("mock-target-page-2")
    expect(firstPage.some((target) => (target.organizations?.length ?? 0) >= 2)).toBe(true)
    expect(firstPage.some((target) => (target.organizations?.some((org) => org.name.length >= 24) ?? false))).toBe(true)
    expect(firstPage.some((target) => target.name.length >= 18)).toBe(true)
    expect(firstPage.some((target) => !target.lastScannedAt)).toBe(true)
    expect(firstPage.some((target) => target.type === "ip")).toBe(true)
    expect(firstPage.some((target) => target.type === "cidr")).toBe(true)
  })

  it("keeps the full mock set varied enough for target-type and relationship coverage", () => {
    expect(mockTargets.some((target) => target.type === "domain")).toBe(true)
    expect(mockTargets.some((target) => target.type === "ip")).toBe(true)
    expect(mockTargets.some((target) => target.type === "cidr")).toBe(true)
    expect(mockTargets.some((target) => (target.organizations?.length ?? 0) >= 3)).toBe(true)
  })

  it("keeps every target detail website summary aligned with the target websites list", () => {
    for (const { id: targetId } of mockTargets) {
      const target = getMockTargetById(targetId)
      const websites = getMockWebsites({ targetId, page: 1, pageSize: 10 })

      expect(target?.summary.websites).toBe(websites.total)
    }
  })

  it("supports mock target deletion for table destructive-action smoke flows", () => {
    const target = {
      id: 998001,
      name: "delete-smoke.example.com",
      type: "domain" as const,
      description: "Temporary target for deletion smoke coverage",
      createdAt: "2026-06-07T00:00:00Z",
      organizations: [],
    }

    mockTargets.push(target)

    try {
      expect(bulkDeleteMockTargets([target.id])).toEqual({ deletedCount: 1 })
      expect(mockTargets.some((item) => item.id === target.id)).toBe(false)
      expect(deleteMockTarget(target.id)).toBe(false)
    } finally {
      deleteMockTarget(target.id)
    }
  })
})
