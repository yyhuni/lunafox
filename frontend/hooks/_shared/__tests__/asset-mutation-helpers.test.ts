import { describe, expect, it } from "vitest"
import {
  getAssetDeletedCount,
  resolveAssetBulkCreateToast,
} from "@/hooks/_shared/asset-mutation-helpers"

describe("asset-mutation-helpers", () => {
  it("returns success toast when createdCount is positive", () => {
    const result = resolveAssetBulkCreateToast(3, {
      success: "toast.asset.sample.create.success",
      partial: "toast.asset.sample.create.partialSuccess",
    })

    expect(result).toEqual({
      variant: "success",
      key: "toast.asset.sample.create.success",
      params: { count: 3 },
    })
  })

  it("returns warning toast when createdCount is zero or undefined", () => {
    const result = resolveAssetBulkCreateToast(0, {
      success: "toast.asset.sample.create.success",
      partial: "toast.asset.sample.create.partialSuccess",
    })

    expect(result).toEqual({
      variant: "warning",
      key: "toast.asset.sample.create.partialSuccess",
      params: { success: 0, skipped: 0 },
    })
  })

  it("falls back to zero for deleted count", () => {
    expect(getAssetDeletedCount({ deletedCount: 2 })).toBe(2)
    expect(getAssetDeletedCount({})).toBe(0)
  })
})
