import { describe, expect, it } from "vitest"
import {
  ASSET_SPARKLINE_VIEW_HEIGHT,
  ASSET_SPARKLINE_VIEW_WIDTH,
  buildAssetSparklineAreaPoints,
  buildAssetSparklinePoints,
} from "./overview-asset-sparkline"

function parsePoints(points: string): number[][] {
  return points
    .split(" ")
    .filter(Boolean)
    .map((pair) => pair.split(",").map(Number))
}

describe("buildAssetSparklinePoints", () => {
  it("returns no points for an empty series", () => {
    expect(buildAssetSparklinePoints([])).toBe("")
  })

  it("renders a single value as a flat line across the viewBox", () => {
    expect(parsePoints(buildAssetSparklinePoints([42]))).toEqual([
      [0, ASSET_SPARKLINE_VIEW_HEIGHT / 2],
      [ASSET_SPARKLINE_VIEW_WIDTH, ASSET_SPARKLINE_VIEW_HEIGHT / 2],
    ])
  })

  it("renders a flat series at the vertical center without inventing a slope", () => {
    const points = parsePoints(buildAssetSparklinePoints([7, 7, 7]))

    expect(points).toHaveLength(3)
    expect(new Set(points.map(([, y]) => y)).size).toBe(1)
    expect(points[0][1]).toBe(ASSET_SPARKLINE_VIEW_HEIGHT / 2)
  })

  it("maps the highest and lowest values inside the padded vertical band", () => {
    const points = parsePoints(buildAssetSparklinePoints([1, 3, 2]))

    expect(points[0][1]).toBe(ASSET_SPARKLINE_VIEW_HEIGHT - 2)
    expect(points[1][1]).toBe(2)
    expect(points[2][1]).toBeGreaterThan(points[1][1])
    expect(points[2][1]).toBeLessThan(points[0][1])
  })

  it("closes the area on the viewBox floor", () => {
    const points = buildAssetSparklinePoints([1, 2, 3])
    const area = parsePoints(buildAssetSparklineAreaPoints(points))

    expect(area[0]).toEqual([0, ASSET_SPARKLINE_VIEW_HEIGHT])
    expect(area.at(-1)).toEqual([ASSET_SPARKLINE_VIEW_WIDTH, ASSET_SPARKLINE_VIEW_HEIGHT])
    expect(buildAssetSparklineAreaPoints("")).toBe("")
  })

  it("keeps the first and last x coordinates on the viewBox edges", () => {
    const points = parsePoints(buildAssetSparklinePoints([1, 2, 3, 4]))

    expect(points[0][0]).toBe(0)
    expect(points.at(-1)?.[0]).toBe(ASSET_SPARKLINE_VIEW_WIDTH)
  })
})
