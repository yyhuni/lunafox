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
      {
        pageSize: 5,
        pageToken: "token-page-2",
        filter: 'displayName="foo"',
        orderBy: "displayName",
      },
      { enabled: false }
    )

    expect(resolved).toMatchObject({
      pageSize: 5,
      pageToken: "token-page-2",
      filter: 'displayName="foo"',
      orderBy: "displayName",
      enabled: false,
    })
  })

  it("resolves default target list params without legacy page or type", () => {
    const resolved = resolveTargetsQueryInput()
    expect(resolved).toMatchObject({
      pageSize: 10,
      pageToken: undefined,
      filter: undefined,
      orderBy: undefined,
      enabled: true,
    })
    expect(resolved).not.toHaveProperty("page")
    expect(resolved).not.toHaveProperty("type")
  })

  it("returns compatibility fields without organization filter", () => {
    const response: TargetsResponse = {
      results: [makeTarget(1), makeTarget(2)],
      total: 2,
      totalSize: 2,
      nextPageToken: "next-token",
      page: 1,
      pageSize: 10,
      totalPages: 1,
    }

    const selected = selectTargetsResponse(response, {
      pageSize: 10,
    })

    expect(selected.targets).toHaveLength(2)
    expect(selected.count).toBe(2)
    expect(selected.total).toBe(2)
    expect(selected.totalSize).toBe(2)
    expect(selected.nextPageToken).toBe("next-token")
  })
})
