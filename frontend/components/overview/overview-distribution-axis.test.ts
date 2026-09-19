import { describe, expect, it } from "vitest"
import { getUiSansCanvasFont } from "@/lib/font-stacks"
import { measureTextWidth } from "@/lib/table-utils"
import {
  ASSET_DISTRIBUTION_AXIS_TICK_MARGIN,
  ASSET_DISTRIBUTION_AXIS_TICK_OFFSET,
  resolveAssetDistributionAxisWidth,
} from "./overview-distribution-axis"

const ASSET_DISTRIBUTION_LABELS = {
  zh: ["子域名", "IP", "URL", "站点"],
  en: ["Subdomains", "IP", "Endpoints", "Websites"],
} as const

const AXIS_TICK_FONT = getUiSansCanvasFont({ sizePx: 12 })

describe("resolveAssetDistributionAxisWidth", () => {
  it("sizes the column from the active locale's own labels", () => {
    expect(resolveAssetDistributionAxisWidth(ASSET_DISTRIBUTION_LABELS.zh)).toBe(56)
    expect(resolveAssetDistributionAxisWidth(ASSET_DISTRIBUTION_LABELS.en)).toBe(88)
  })

  it("keeps the column bounded when no label or only a short label is rendered", () => {
    expect(resolveAssetDistributionAxisWidth([])).toBe(56)
    expect(resolveAssetDistributionAxisWidth(["IP"])).toBe(56)
  })

  it("widens the column for a longer label and stops at the column cap", () => {
    expect(resolveAssetDistributionAxisWidth(["IP 地址"])).toBeGreaterThan(
      resolveAssetDistributionAxisWidth(["IP"]),
    )
    expect(resolveAssetDistributionAxisWidth(["一个非常长的资产分类名称"])).toBe(96)
  })

  it("reserves the widest label plus its tick gap, so no label is cut off at the panel edge", () => {
    for (const labels of [ASSET_DISTRIBUTION_LABELS.zh, ASSET_DISTRIBUTION_LABELS.en]) {
      const widestLabel = Math.max(...labels.map((label) => measureTextWidth(label, AXIS_TICK_FONT)))

      expect(resolveAssetDistributionAxisWidth(labels)).toBeGreaterThanOrEqual(
        widestLabel + ASSET_DISTRIBUTION_AXIS_TICK_MARGIN + ASSET_DISTRIBUTION_AXIS_TICK_OFFSET,
      )
    }
  })
})
