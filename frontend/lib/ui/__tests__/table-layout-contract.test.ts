import { describe, expect, it } from "vitest"

import { getFixedTableLayoutTotalWidth } from "@/lib/ui/table-layout-contract"

describe("table layout contract", () => {
  it("sums fixed column sizes into one shared shell width", () => {
    expect(getFixedTableLayoutTotalWidth([
      { key: "select", size: 40 },
      { key: "name", size: 300 },
      { key: "actions", size: 56 },
    ])).toBe(396)
  })

  it("fast-fails when a fixed table layout is incomplete", () => {
    expect(() => getFixedTableLayoutTotalWidth([])).toThrow(
      "getFixedTableLayoutTotalWidth requires at least one column."
    )
    expect(() => getFixedTableLayoutTotalWidth([
      { key: "broken", size: 0 },
    ])).toThrow('Table column "broken" requires a positive finite size.')
  })
})
