import { describe, expect, it } from "vitest"

import { virtualTableScrollPadding } from "@/components/shared/data-table/virtual-table-padding"

describe("virtualTableScrollPadding", () => {
  it("keeps a 10000-row window's prefix and suffix equal to the full list", () => {
    const rowSize = 41
    const scrollMargin = 41
    const count = 10_000
    const totalSize = count * rowSize
    const startIndex = 5_000
    const endIndex = 5_024
    const virtualRows = Array.from({ length: endIndex - startIndex + 1 }, (_, offset) => {
      const index = startIndex + offset
      const start = scrollMargin + index * rowSize
      return { start, end: start + rowSize, size: rowSize }
    })

    const padding = virtualTableScrollPadding(virtualRows, totalSize, scrollMargin)
    const visibleSize = virtualRows.reduce((sum, row) => sum + row.size, 0)

    expect(padding.paddingTop).toBe(startIndex * rowSize)
    expect(padding.paddingTop + visibleSize + padding.paddingBottom).toBe(totalSize)
  })

  it("omits the leading pad when the window starts at the first row", () => {
    const padding = virtualTableScrollPadding(
      [{ start: 41, end: 82, size: 41 }],
      410,
      41,
    )

    expect(padding).toEqual({ paddingTop: 0, paddingBottom: 369 })
  })
})
