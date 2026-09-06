import { describe, expect, it } from "vitest"

import { getBadgeMinWidthPx, getColumnHeaderMinWidthPx } from "@/lib/table-utils"

describe("table-utils", () => {
  it("gives sortable headers a larger floor than plain text-only headers", () => {
    const plainWidth = getColumnHeaderMinWidthPx("虚拟主机", {
      includeSortIcon: false,
    })
    const sortableWidth = getColumnHeaderMinWidthPx("虚拟主机")

    expect(sortableWidth).toBeGreaterThan(plainWidth)
  })

  it("reserves enough width for compact Chinese sortable headers", () => {
    expect(getColumnHeaderMinWidthPx("状态码")).toBeGreaterThan(90)
  })

  it("gives longer badge labels a larger floor than short ones", () => {
    const shortBadgeWidth = getBadgeMinWidthPx("json")
    const longBadgeWidth = getBadgeMinWidthPx("application/javascript")

    expect(longBadgeWidth).toBeGreaterThan(shortBadgeWidth)
  })
})
