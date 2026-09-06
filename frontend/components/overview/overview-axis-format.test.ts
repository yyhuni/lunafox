import { describe, expect, it } from "vitest"
import { formatAssetTrendAxisTick, formatOverviewCompactNumber, formatOverviewDateTick, formatOverviewSignedChange } from "./overview-axis-format"

describe("formatAssetTrendAxisTick", () => {
  it("keeps values below ten-thousand unscaled", () => {
    expect(formatAssetTrendAxisTick(0, 99, "en")).toBe("0")
    expect(formatAssetTrendAxisTick(80, 99, "en")).toBe("80")
    expect(formatAssetTrendAxisTick(999, 999, "en")).toBe("999")
    expect(formatAssetTrendAxisTick(4823, 4823, "en")).toBe("4,823")
  })

  it("uses locale-aware compact units when the active axis range reaches ten-thousand", () => {
    expect(formatAssetTrendAxisTick(6000, 24000, "zh")).toBe("6000")
    expect(formatAssetTrendAxisTick(12000, 24000, "zh")).toBe("1.2万")
    expect(formatAssetTrendAxisTick(24000, 24000, "zh")).toBe("2.4万")
    expect(formatAssetTrendAxisTick(24000, 24000, "en")).toBe("24K")
  })

  it("uses locale-aware compact units for hundred-million-scale axes", () => {
    expect(formatAssetTrendAxisTick(50000000, 120000000, "zh")).toBe("5000万")
    expect(formatAssetTrendAxisTick(120000000, 120000000, "zh")).toBe("1.2亿")
    expect(formatAssetTrendAxisTick(120000000, 120000000, "en")).toBe("120M")
  })
})

describe("formatOverviewCompactNumber", () => {
  it("keeps dashboard metric numbers compact without scaling small values", () => {
    expect(formatOverviewCompactNumber(89, "en")).toBe("89")
    expect(formatOverviewCompactNumber(9999, "en")).toBe("9,999")
  })

  it("uses locale-aware compact units for large dashboard metric numbers", () => {
    expect(formatOverviewCompactNumber(12345, "zh")).toBe("1.2万")
    expect(formatOverviewCompactNumber(1234567, "zh")).toBe("123.5万")
    expect(formatOverviewCompactNumber(120000000, "zh")).toBe("1.2亿")
    expect(formatOverviewCompactNumber(120000000, "en")).toBe("120M")
  })
})

describe("formatOverviewSignedChange", () => {
  it("keeps a visible sign for positive, negative, and unchanged metrics", () => {
    expect(formatOverviewSignedChange(942, "zh")).toBe("+942")
    expect(formatOverviewSignedChange(-942, "en")).toBe("-942")
    expect(formatOverviewSignedChange(0, "en")).toBe("+0")
  })
})

describe("formatOverviewDateTick", () => {
  it("keeps date-only trend labels compact and timezone-stable", () => {
    expect(formatOverviewDateTick("2026-07-12", "zh")).toMatch(/7.*12/)
    expect(formatOverviewDateTick("2026-07-12", "en")).toMatch(/7.*12/)
  })

  it("keeps an invalid source value visible instead of producing an invalid date label", () => {
    expect(formatOverviewDateTick("not-a-date", "zh")).toBe("not-a-date")
  })
})
