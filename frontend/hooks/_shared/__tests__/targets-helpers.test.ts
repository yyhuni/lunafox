import { describe, expect, it } from "vitest"
import {
  resolveTargetsQueryInput,
  selectTargetsResponse,
} from "@/hooks/_shared/targets-helpers"
import type { TargetsResponse, Target } from "@/types/target.types"

const makeTarget = (id: number, orgIds: number[] = []): Target => ({
  id,
  name: `target-${id}`,
  type: "domain",
  createdAt: "2026-01-01T00:00:00Z",
  organizations: orgIds.map((orgId) => ({ id: orgId, name: `org-${orgId}` })),
})

describe("targets helpers", () => {
  it("resolves object params with options", () => {
    const resolved = resolveTargetsQueryInput(
      { page: 2, pageSize: 5, organizationId: 9, filter: "foo" },
      { enabled: false }
    )

    expect(resolved).toMatchObject({
      page: 2,
      pageSize: 5,
      organizationId: 9,
      filter: "foo",
      enabled: false,
      type: undefined,
    })
  })

  it("resolves positional params", () => {
    const resolved = resolveTargetsQueryInput(3, 20, "ip", "bar")
    expect(resolved).toMatchObject({
      page: 3,
      pageSize: 20,
      organizationId: undefined,
      filter: "bar",
      type: "ip",
      enabled: true,
    })
  })

  it("filters targets by organization when requested", () => {
    const response: TargetsResponse = {
      results: [makeTarget(1, [1]), makeTarget(2, [2])],
      total: 2,
      page: 1,
      pageSize: 10,
      totalPages: 1,
    }

    const selected = selectTargetsResponse(response, {
      page: 1,
      pageSize: 10,
      organizationId: 2,
    })

    expect(selected.targets).toHaveLength(1)
    expect(selected.targets[0]?.id).toBe(2)
    expect(selected.count).toBe(1)
    expect(selected.total).toBe(1)
  })

  it("returns compatibility fields without organization filter", () => {
    const response: TargetsResponse = {
      results: [makeTarget(1), makeTarget(2)],
      total: 2,
      page: 1,
      pageSize: 10,
      totalPages: 1,
    }

    const selected = selectTargetsResponse(response, {
      page: 1,
      pageSize: 10,
    })

    expect(selected.targets).toHaveLength(2)
    expect(selected.count).toBe(2)
    expect(selected.total).toBe(2)
  })
})
