import { describe, expect, it } from "vitest"
import {
  getSubdomainBatchDeleteCount,
  getSubdomainBatchDeleteFromOrgCount,
  resolveSubdomainCreateToast,
} from "@/hooks/_shared/subdomain-mutation-helpers"

describe("subdomain-mutation-helpers", () => {
  it("returns warning toast when there are skipped/existed items", () => {
    const result = resolveSubdomainCreateToast({
      createdCount: 2,
      existedCount: 1,
      skippedCount: 1,
    })

    expect(result).toEqual({
      variant: "warning",
      key: "toast.asset.subdomain.create.partialSuccess",
      params: {
        success: 2,
        skipped: 2,
      },
    })
  })

  it("returns success toast when no skipped/existed items", () => {
    const result = resolveSubdomainCreateToast({
      createdCount: 3,
      existedCount: 0,
      skippedCount: 0,
    })

    expect(result).toEqual({
      variant: "success",
      key: "toast.asset.subdomain.create.success",
      params: {
        count: 3,
      },
    })
  })

  it("computes delete counts with fallback", () => {
    expect(getSubdomainBatchDeleteCount({ deletedCount: 4 })).toBe(4)
    expect(getSubdomainBatchDeleteCount({})).toBe(0)
    expect(getSubdomainBatchDeleteFromOrgCount({ successCount: 2 })).toBe(2)
    expect(getSubdomainBatchDeleteFromOrgCount({})).toBe(0)
  })
})
