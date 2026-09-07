"use client"

import { useEffect, useState } from "react"
import { useReducedMotion } from "framer-motion"
import { useLocale, useTranslations } from "next-intl"
import { Bar, BarChart, Cell, LabelList, XAxis, YAxis, type LabelProps } from "recharts"
import { useAssetStatistics, useStatisticsHistory } from "@/hooks/use-overview"
import { semanticIcons } from "@/components/icons"
import { OverviewAreaChart } from "@/components/overview/overview-area-chart"
import { OverviewNumberTicker } from "@/components/overview/overview-number-ticker"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import {
  OVERVIEW_ASSET_CHART_SHELL_CLASS,
  OVERVIEW_ASSET_DISTRIBUTION_CHART_CLASS,
  OVERVIEW_ASSET_METRIC_CLASS,
  OVERVIEW_ASSET_METRICS_GRID_CLASS,
  OVERVIEW_ASSET_OVERVIEW_GRID_CLASS,
  OverviewSectionPanel,
} from "@/components/overview/overview-section-layouts"
import { formatAssetTrendAxisTick, formatOverviewDateTick, formatOverviewSignedChange } from "@/components/overview/overview-axis-format"
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart"
import type { AssetDistributionChartSeries } from "@/lib/chart-config"
import { getTrendToneTextClass } from "@/lib/status-config"
import { normalizeError } from "@/lib/errors/normalize-error"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

const AssetIcon = semanticIcons.concept.asset
const TargetIcon = semanticIcons.concept.target

const ASSET_CHART_MARGIN = { top: 12, right: 0, left: 0, bottom: 0 }
const ASSET_CHART_X_AXIS_PADDING = { left: 0, right: 16 }
const ASSET_DISTRIBUTION_Y_AXIS_WIDTH = 64
const ASSET_DISTRIBUTION_BASE_COLOR = "var(--foreground)"
const ASSET_DISTRIBUTION_MIN_OPACITY = 0.32
const ASSET_DISTRIBUTION_MAX_OPACITY = 0.68
const ASSET_CHART_DOT = {
  fill: "var(--card)",
  r: 3,
  stroke: "var(--primary)",
  strokeWidth: 2,
}
const ASSET_CHART_TOOLTIP_STYLE = {
  background: "var(--card)",
  border: "1px solid var(--border)",
  borderRadius: 8,
  boxShadow: "none",
}

function AssetDistributionValueLabel({ value, viewBox }: LabelProps) {
  if (
    value === undefined ||
    !viewBox ||
    !("x" in viewBox) ||
    viewBox.x === undefined ||
    viewBox.y === undefined ||
    viewBox.width === undefined ||
    viewBox.height === undefined
  ) {
    return null
  }

  return (
    <text
      className="fill-foreground"
      dominantBaseline="middle"
      fontSize={11}
      x={viewBox.x + viewBox.width + 8}
      y={viewBox.y + viewBox.height / 2}
    >
      {String(value)}
    </text>
  )
}

export function OverviewAssetOverview() {
  const t = useTranslations("overview.operational.assets")
  const tAsset = useTranslations("overview.lazySections.assetStatus")
  const locale = useLocale()
  const assets = useAssetStatistics()
  const history = useStatisticsHistory(7)
  const prefersReducedMotion = useReducedMotion()
  const [isTrendAnimationActive, setIsTrendAnimationActive] = useState(false)
  const assetTypes: Array<{
    key: string
    series: AssetDistributionChartSeries
    label: string
    value: number
  }> = [
    { key: "subdomains", series: "subdomain", label: tAsset("tabs.subdomains"), value: assets.data?.totalSubdomains ?? 0 },
    { key: "ips", series: "ip", label: tAsset("tabs.ips"), value: assets.data?.totalIps ?? 0 },
    { key: "endpoints", series: "endpoint", label: tAsset("tabs.endpoints"), value: assets.data?.totalEndpoints ?? 0 },
    { key: "websites", series: "website", label: tAsset("tabs.websites"), value: assets.data?.totalWebsites ?? 0 },
  ]
  const distributionTotal = assetTypes.reduce((sum, item) => sum + item.value, 0)
  const assetDistributionConfig = {
    value: {
      label: t("assets"),
    },
    subdomain: {
      label: tAsset("tabs.subdomains"),
      color: ASSET_DISTRIBUTION_BASE_COLOR,
    },
    ip: {
      label: tAsset("tabs.ips"),
      color: ASSET_DISTRIBUTION_BASE_COLOR,
    },
    endpoint: {
      label: tAsset("tabs.endpoints"),
      color: ASSET_DISTRIBUTION_BASE_COLOR,
    },
    website: {
      label: tAsset("tabs.websites"),
      color: ASSET_DISTRIBUTION_BASE_COLOR,
    },
  } satisfies ChartConfig
  const maxDistributionValue = Math.max(...assetTypes.map((item) => item.value), 1)
  const assetDistributionData = assetTypes.map((item) => {
    const percent = distributionTotal > 0 ? Math.round((item.value / distributionTotal) * 100) : 0

    return {
      ...item,
      displayValue: `${item.value.toLocaleString(locale)} (${percent}%)`,
      fill: ASSET_DISTRIBUTION_BASE_COLOR,
      fillOpacity:
        ASSET_DISTRIBUTION_MIN_OPACITY +
        (item.value / maxDistributionValue) *
          (ASSET_DISTRIBUTION_MAX_OPACITY - ASSET_DISTRIBUTION_MIN_OPACITY),
    }
  })
  const historyData = (history.data ?? []).map((item) => ({
    date: item.date,
    totalAssets: item.totalAssets,
  }))
  useEffect(() => {
    if (!historyData.length || prefersReducedMotion !== false) {
      setIsTrendAnimationActive(false)
      return
    }

    setIsTrendAnimationActive(true)
  }, [historyData.length, prefersReducedMotion])
  const maxHistoryValue = Math.max(...historyData.map((item) => item.totalAssets), assets.data?.totalAssets ?? 0)
  const assetChange = assets.data?.changeAssets ?? 0
  const targetChange = assets.data?.changeTargets ?? 0
  const assetChangeTone = assetChange > 0 ? "positive" : assetChange < 0 ? "negative" : "neutral"
  const targetChangeTone = targetChange > 0 ? "positive" : targetChange < 0 ? "negative" : "neutral"
  return (
    <OverviewSectionPanel
      contentClassName="pt-1"
    >
      {assets.data ? (
        <div className={OVERVIEW_ASSET_OVERVIEW_GRID_CLASS}>
          <div className="min-w-0 space-y-6">
            <div className={OVERVIEW_ASSET_METRICS_GRID_CLASS}>
              <div className={OVERVIEW_ASSET_METRIC_CLASS}>
                <div className="flex items-center gap-2">
                  <AssetIcon aria-hidden="true" className="size-4 text-muted-foreground" />
                  <span className={textRole.caption}>{t("assets")}</span>
                </div>
                <span className={textRole.metricValueDisplay}>
                  <OverviewNumberTicker value={assets.data?.totalAssets ?? 0} locale={locale} />
                </span>
                <span className={textRole.caption}>
                  {t("change")}{" "}
                  <span className={getTrendToneTextClass(assetChangeTone)}>{formatOverviewSignedChange(assetChange, locale)}</span>
                </span>
              </div>
              <div className={OVERVIEW_ASSET_METRIC_CLASS}>
                <div className="flex items-center gap-2">
                  <TargetIcon aria-hidden="true" className="size-4 text-muted-foreground" />
                  <span className={textRole.caption}>{t("targets")}</span>
                </div>
                <span className={textRole.metricValueDisplay}>
                  <OverviewNumberTicker value={assets.data?.totalTargets ?? 0} locale={locale} />
                </span>
                <span className={textRole.caption}>
                  {t("change")}{" "}
                  <span className={getTrendToneTextClass(targetChangeTone)}>{formatOverviewSignedChange(targetChange, locale)}</span>
                </span>
              </div>
            </div>

            <div
              aria-label={assetDistributionData.map((item) => `${item.label} ${item.displayValue}`).join(", ")}
              className="min-w-0"
              role="img"
            >
              <ChartContainer
                config={assetDistributionConfig}
                className={OVERVIEW_ASSET_DISTRIBUTION_CHART_CLASS}
              >
                <BarChart
                  accessibilityLayer
                  data={assetDistributionData}
                  layout="vertical"
                  margin={{ top: 2, right: 92, left: -14, bottom: 2 }}
                >
                  <XAxis dataKey="value" type="number" hide />
                  <YAxis
                    axisLine={false}
                    dataKey="label"
                    tick={{ dx: -36, textAnchor: "start" }}
                    tickLine={false}
                    tickMargin={8}
                    type="category"
                    width={ASSET_DISTRIBUTION_Y_AXIS_WIDTH}
                  />
                  <ChartTooltip
                    cursor={false}
                    content={<ChartTooltipContent hideLabel nameKey="series" />}
                  />
                  <Bar
                    barSize={20}
                    dataKey="value"
                    radius={2}
                  >
                    {assetDistributionData.map((item) => (
                      <Cell key={item.key} fill={item.fill} fillOpacity={item.fillOpacity} />
                    ))}
                    <LabelList
                      content={AssetDistributionValueLabel}
                      dataKey="displayValue"
                      position="right"
                    />
                  </Bar>
                </BarChart>
              </ChartContainer>
            </div>
          </div>

          <div className="min-w-0 border-t pt-5 xl:border-l xl:border-t-0 xl:pl-6 xl:pt-0">
            <div className="flex items-center justify-between gap-3">
              <h3 className={textRole.sectionTitle}>{t("trend")}</h3>
              <span className={textRole.caption}>{history.data?.length ?? 0}d</span>
            </div>
            {history.isError || !historyData.length ? (
              <div className={cn("flex items-center justify-center text-center", OVERVIEW_ASSET_CHART_SHELL_CLASS)}>
                <p className={textRole.bodySubtle}>{tAsset("trendTitle")}</p>
              </div>
            ) : (
              <div
                className={cn(
                  "overview-chart-enter mt-4",
                  isTrendAnimationActive && "overview-chart-draw",
                  OVERVIEW_ASSET_CHART_SHELL_CLASS,
                )}
                onAnimationEnd={(event) => {
                  if (event.animationName === "overview-chart-path-reveal") {
                    setIsTrendAnimationActive(false)
                  }
                }}
              >
                <OverviewAreaChart
                  data={historyData}
                  dataKey="totalAssets"
                  gradientId="operationalAssetTrendFill"
                  margin={ASSET_CHART_MARGIN}
                  tooltipFormatter={(value) => [value.toLocaleString(locale), t("assets")]}
                  tooltipLabelFormatter={(label) => String(label)}
                  xAxisDataKey="date"
                  xAxisInterval={0}
                  xAxisMinTickGap={0}
                  xAxisPadding={ASSET_CHART_X_AXIS_PADDING}
                  xAxisTickFontSize={11}
                  xAxisTickFormatter={(value) => formatOverviewDateTick(value, locale)}
                  xAxisTickMargin={8}
                  yAxisTickFormatter={(value) => Number(value) === 0 ? "" : formatAssetTrendAxisTick(Number(value), maxHistoryValue, locale)}
                  yAxisWidth={44}
                  contentStyle={ASSET_CHART_TOOLTIP_STYLE}
                  dot={ASSET_CHART_DOT}
                  activeDotRadius={5}
                  isAnimationActive={false}
                />
              </div>
            )}
          </div>
        </div>
      ) : assets.error ? (
        <AppErrorState
          error={normalizeError(assets.error, { notFoundKind: "unexpected-error" })}
          title={t("loadFailed")}
          description={t("loadFailedDescription")}
          onRetry={assets.refetch}
          variant="section"
          className={cn(OVERVIEW_ASSET_CHART_SHELL_CLASS, "py-4")}
        />
      ) : (
        <div className={cn("flex items-center justify-center text-center", OVERVIEW_ASSET_CHART_SHELL_CLASS)}>
          <p className={textRole.bodySubtle}>{t("unavailable")}</p>
        </div>
      )}
    </OverviewSectionPanel>
  )
}
