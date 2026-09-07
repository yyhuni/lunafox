import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const sourcePath = path.resolve(process.cwd(), "lib/chart-config.ts")

describe("chart-config contract", () => {
  it("exports shared chart ownership helpers for categorical overviews", () => {
    expect(existsSync(sourcePath)).toBe(true)
    const source = readFileSync(sourcePath, "utf8")
    expect(source).toContain("export const CHART_PALETTE")
    expect(source).toContain("export const ASSET_DISTRIBUTION_CHART_SERIES")
    expect(source).toContain("export function getChartPaletteColor")
    expect(source).toContain("export function getAssetDistributionChartColor")
  })

  it("exports shared agent architecture role helpers", () => {
    const source = readFileSync(sourcePath, "utf8")
    expect(source).toContain("export type ArchitectureFlowRoleAccent")
    expect(source).toContain("export function getArchitectureFlowRoleColor")
    expect(source).toContain("export function getArchitectureFlowRoleIconClassName")
    expect(source).toContain("export function getArchitectureFlowRoleDotClassName")
  })
})
