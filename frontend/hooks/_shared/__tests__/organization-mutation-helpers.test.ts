import { describe, expect, it, vi } from "vitest"
import {
  applyOrganizationOptimisticDelete,
  getOrganizationDeleteToastId,
  invalidateOrganizationTargets,
  ORGANIZATION_BATCH_DELETE_TOAST_ID,
  ORGANIZATION_QUERY_KEY,
  rollbackOrganizationQueries,
  TARGET_QUERY_KEY,
} from "@/hooks/_shared/organization-mutation-helpers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

describe("organization-mutation-helpers", () => {
  it("derives toast ids", () => {
    expect(getOrganizationDeleteToastId(9)).toBe("delete-9")
    expect(ORGANIZATION_BATCH_DELETE_TOAST_ID).toBe("batch-delete")
  })

  it("optimistically filters organizations and can rollback", () => {
    const queryClient = createTestQueryClient()
    queryClient.setQueryData(ORGANIZATION_QUERY_KEY, {
      organizations: [
        { id: 1, name: "A" },
        { id: 2, name: "B" },
      ],
    })

    const snapshot = applyOrganizationOptimisticDelete(queryClient, [1])
    const updated = queryClient.getQueryData<{ organizations?: Array<{ id: number }> }>(
      ORGANIZATION_QUERY_KEY
    )

    expect(updated?.organizations?.map((org) => org.id)).toEqual([2])

    rollbackOrganizationQueries(queryClient, snapshot)
    const restored = queryClient.getQueryData<{ organizations?: Array<{ id: number }> }>(
      ORGANIZATION_QUERY_KEY
    )
    expect(restored?.organizations?.map((org) => org.id)).toEqual([1, 2])
  })

  it("invalidates organization and target queries", async () => {
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    await invalidateOrganizationTargets(queryClient)

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ORGANIZATION_QUERY_KEY })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: TARGET_QUERY_KEY })
  })
})
