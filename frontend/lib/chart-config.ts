export const CHART_PALETTE = {
  chart1: "var(--chart-1)",
  chart2: "var(--chart-2)",
  chart3: "var(--chart-3)",
  chart4: "var(--chart-4)",
  chart5: "var(--chart-5)",
} as const

export type ChartPaletteKey = keyof typeof CHART_PALETTE
export type AssetDistributionChartSeries = "subdomain" | "ip" | "endpoint" | "website"
export type ArchitectureFlowRoleAccent = "agent" | "engine"

export const ASSET_DISTRIBUTION_CHART_SERIES = [
  "subdomain",
  "ip",
  "endpoint",
  "website",
] as const satisfies readonly AssetDistributionChartSeries[]

const ASSET_DISTRIBUTION_CHART_PALETTE: Record<AssetDistributionChartSeries, ChartPaletteKey> = {
  subdomain: "chart1",
  ip: "chart2",
  endpoint: "chart3",
  website: "chart4",
}

const ARCHITECTURE_FLOW_ROLE_PALETTE: Record<ArchitectureFlowRoleAccent, ChartPaletteKey> = {
  agent: "chart2",
  engine: "chart1",
}

export function getChartPaletteColor(key: ChartPaletteKey): string {
  return CHART_PALETTE[key]
}

export function getAssetDistributionChartColor(series: AssetDistributionChartSeries): string {
  return getChartPaletteColor(ASSET_DISTRIBUTION_CHART_PALETTE[series])
}

export function getArchitectureFlowRoleColor(role: ArchitectureFlowRoleAccent): string {
  return getChartPaletteColor(ARCHITECTURE_FLOW_ROLE_PALETTE[role])
}

export function getArchitectureFlowRoleIconClassName(role: ArchitectureFlowRoleAccent): string {
  return role === "agent"
    ? "text-[color:var(--chart-2)]"
    : "text-[color:var(--chart-1)]"
}

export function getArchitectureFlowRoleDotClassName(role: ArchitectureFlowRoleAccent): string {
  return role === "agent"
    ? "bg-[color:var(--chart-2)]"
    : "bg-[color:var(--chart-1)]"
}
