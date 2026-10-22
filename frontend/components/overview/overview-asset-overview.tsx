"use client"

import { useEffect, useState } from "react"
import { useReducedMotion } from "framer-motion"
import { useLocale, useTranslations } from "next-intl"
import { useAssetStatistics, useStatisticsHistory } from "@/hooks/use-overview"
import { semanticIcons } from "@/components/icons"
import { OverviewAreaChart } from "@/components/overview/overview-area-chart"
import {
  ASSET_SPARKLINE_VIEW_HEIGHT,
  ASSET_SPARKLINE_VIEW_WIDTH,
  buildAssetSparklineAreaPoints,
  buildAssetSparklinePoints,
} from "@/components/overview/overview-asset-sparkline"
import { OverviewNumberTicker } from "@/components/overview/overview-number-ticker"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import {
  OVERVIEW_ASSET_CHART_SHELL_CLASS,
  OVERVIEW_ASSET_DISTRIBUTION_COLUMNS_CLASS,
  OVERVIEW_ASSET_DISTRIBUTION_HEADER_GRID_CLASS,
  OVERVIEW_ASSET_DISTRIBUTION_LIST_CLASS,
  OVERVIEW_ASSET_DISTRIBUTION_ROW_CLASS,
  OVERVIEW_ASSET_METRIC_CLASS,
  OVERVIEW_ASSET_METRICS_GRID_CLASS,
  OVERVIEW_ASSET_OVERVIEW_GRID_CLASS,
  OVERVIEW_ASSET_SPARKLINE_CLASS,
  OVERVIEW_ASSET_TREND_CHART_CLASS,
  OVERVIEW_ASSET_TREND_PANEL_CLASS,
  OverviewSectionPanel,
} from "@/components/overview/overview-section-layouts"
import { formatAssetTrendAxisTick, formatOverviewDateTick, formatOverviewSignedChange } from "@/components/overview/overview-axis-format"
import type { AssetDistributionChartSeries } from "@/lib/chart-config"
import { getTrendToneTextClass, type TrendTone } from "@/lib/status-config"
import { normalizeError } from "@/lib/errors/normalize-error"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import { COMPACT_SECTION_STACK_CLASS } from "@/components/shared/layout/page-shell-density"
import type { StatisticsHistoryItem } from "@/types/overview.types"

const AssetIcon = semanticIcons.concept.asset
const TargetIcon = semanticIcons.concept.target

const ASSET_CHART_MARGIN = { top: 12, right: 0, left: 0, bottom: 0 }
const ASSET_CHART_X_AXIS_PADDING = { left: 0, right: 16 }
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
// The row trend, the seven-day delta, and the headline count all describe the
// same asset type, so they must read from the matching statistics field.
const ASSET_TYPE_HISTORY_KEYS = {
  subdomain: "totalSubdomains",
  ip: "totalIps",
  endpoint: "totalEndpoints",
  website: "totalWebsites",
} as const satisfies Record<AssetDistributionChartSeries, keyof StatisticsHistoryItem>

function AssetSparkline({ points }: { points: string }) {
  return (
    <svg
      aria-hidden="true"
      className={OVERVIEW_ASSET_SPARKLINE_CLASS}
      preserveAspectRatio="none"
      viewBox={`0 0 ${ASSET_SPARKLINE_VIEW_WIDTH} ${ASSET_SPARKLINE_VIEW_HEIGHT}`}
    >
      {points ? (
        <>
          <polygon fill="currentColor" fillOpacity={0.18} points={buildAssetSparklineAreaPoints(points)} />
          <polyline
            fill="none"
            points={points}
            stroke="currentColor"
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            vectorEffect="non-scaling-stroke"
          />
        </>
      ) : null}
    </svg>
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
  const historyItems = history.data ?? []
  const assetDistributionRows = assetTypes.map((item) => {
    const seriesValues = historyItems.map((entry) => Number(entry[ASSET_TYPE_HISTORY_KEYS[item.series]] ?? 0))
    const hasHistory = seriesValues.length >= 2
    const sevenDayChange = hasHistory
      ? (seriesValues.at(-1) ?? 0) - (seriesValues.at(0) ?? 0)
      : 0
    const percent = distributionTotal > 0 ? Math.round((item.value / distributionTotal) * 100) : 0
    const changeTone: TrendTone = sevenDayChange > 0 ? "positive" : sevenDayChange < 0 ? "negative" : "neutral"

    return {
      ...item,
      percent,
      hasHistory,
      sevenDayChange,
      changeTone,
      sparklinePoints: buildAssetSparklinePoints(seriesValues),
    }
  })
  const distributionLabel = assetDistributionRows
    .map((row) => {
      const valueLabel = `${row.label} ${row.value.toLocaleString(locale)} (${row.percent}%)`

      return row.hasHistory
        ? `${valueLabel}, ${t("newIn7d")} ${formatOverviewSignedChange(row.sevenDayChange, locale)}`
        : valueLabel
    })
    .join("; ")
  const historyData = historyItems.map((item) => ({
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
          <div className={cn("min-w-0", COMPACT_SECTION_STACK_CLASS)}>
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
              aria-label={distributionLabel}
              className="min-w-0"
              role="img"
            >
              <div className={OVERVIEW_ASSET_DISTRIBUTION_HEADER_GRID_CLASS}>
                <span className={textRole.tableHeader}>{t("type")}</span>
                <span className={cn(textRole.tableHeader, "text-right")}>{t("count")}</span>
                <span className={cn(textRole.tableHeader, "text-right")}>{t("share")}</span>
                <span className={cn(textRole.tableHeader, "text-right")}>{t("newIn7d")}</span>
              </div>
              <div className={OVERVIEW_ASSET_DISTRIBUTION_LIST_CLASS}>
                {assetDistributionRows.map((row) => (
                  <div key={row.key} className={OVERVIEW_ASSET_DISTRIBUTION_ROW_CLASS}>
                    <div className={OVERVIEW_ASSET_DISTRIBUTION_COLUMNS_CLASS}>
                      <span className={cn("truncate", textRole.metadataLabel)}>{row.label}</span>
                      <span className={cn(textRole.metadataValueStrong, "text-right tabular-nums")}>
                        {row.value.toLocaleString(locale)}
                      </span>
                      <span className={cn(textRole.metadataLabel, "text-right tabular-nums")}>{row.percent}%</span>
                      <span
                        className={cn(
                          textRole.metadataValueStrong,
                          "text-right tabular-nums",
                          row.hasHistory ? getTrendToneTextClass(row.changeTone) : "text-muted-foreground",
                        )}
                      >
                        {row.hasHistory
                          ? row.sevenDayChange === 0 ? "0" : formatOverviewSignedChange(row.sevenDayChange, locale)
                          : "—"}
                      </span>
                    </div>
                    <AssetSparkline points={row.sparklinePoints} />
                  </div>
                ))}
              </div>
            </div>
          </div>

          <div className={OVERVIEW_ASSET_TREND_PANEL_CLASS}>
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
                  "overview-chart-enter",
                  OVERVIEW_ASSET_TREND_CHART_CLASS,
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
