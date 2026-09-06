"use client"

import Link from "next/link"
import { useMemo, useState } from "react"
import { useLocale, useTranslations } from "next-intl"
import { Cell, Pie, PieChart, Sector } from "recharts"
import type { PieSectorDataItem } from "recharts/types/polar/Pie"
import { useAssetStatistics, useStatisticsHistory } from "@/hooks/use-overview"
import { useVulnerabilityStats } from "@/hooks/use-vulnerabilities"
import { IconChevronRight, semanticIcons } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { OverviewAreaChart } from "@/components/overview/overview-area-chart"
import {
  OVERVIEW_RISK_LEGEND_CLASS,
  OVERVIEW_RISK_LEGEND_ROW_CLASS,
  OVERVIEW_RISK_RING_CLASS,
  OVERVIEW_RISK_RING_LEGEND_CLASS,
  OVERVIEW_RISK_TREND_CHART_CLASS,
  OVERVIEW_RISK_SUMMARY_DETAIL_ROW_CLASS,
  OVERVIEW_RISK_SUMMARY_DETAILS_CLASS,
  OVERVIEW_RISK_SUMMARY_GRID_CLASS,
  OVERVIEW_RISK_SUMMARY_METRICS_CLASS,
  OVERVIEW_TREND_HEADER_CLASS,
  OverviewSectionPanel,
} from "@/components/overview/overview-section-layouts"
import { OverviewNumberTicker } from "@/components/overview/overview-number-ticker"
import { formatOverviewDateTick, formatOverviewSignedChange } from "@/components/overview/overview-axis-format"
import { Skeleton } from "@/components/ui/skeleton"
import { getSeverityColor, SEVERITY_LEVELS, SEVERITY_TEXT_CLASSNAMES } from "@/lib/severity-config"
import { getTrendToneColorVar, getTrendToneTextClass } from "@/lib/status-config"
import { normalizeError } from "@/lib/errors/normalize-error"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

const RISK_TREND_CHART_MARGIN = { top: 8, right: 0, bottom: 0, left: 0 }
const RISK_TREND_TOOLTIP_STYLE = {
  background: "var(--card)",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius)",
  boxShadow: "none",
}

const RiskIcon = semanticIcons.status.risk

function renderActiveRiskSector({ outerRadius = 0, ...props }: PieSectorDataItem) {
  const expandedOuterRadius = typeof outerRadius === "number" ? outerRadius + 7 : outerRadius

  return <Sector {...props} outerRadius={expandedOuterRadius} />
}

export function OverviewRiskSummary({ className }: { className?: string } = {}) {
  const t = useTranslations("overview.operational.risk")
  const tSeverity = useTranslations("severity")
  const locale = useLocale()
  const assets = useAssetStatistics()
  const { data } = assets
  const riskHistory = useStatisticsHistory(7)
  const vulnerabilityStats = useVulnerabilityStats()
  // `changeVulns` is a net delta; a decrease must not be presented as newly discovered risk.
  const newVulnerabilityCount = Math.max(data?.changeVulns ?? 0, 0)
  const newVulnerabilityTone = newVulnerabilityCount > 0 ? "negative" : "neutral"
  const auditStats = vulnerabilityStats.data
  const [activeRiskIndex, setActiveRiskIndex] = useState<number>()
  const chartConfig = useMemo(() => ({
    count: {
      label: t("open"),
    },
    ...Object.fromEntries(
      SEVERITY_LEVELS.map((severity) => [
        severity,
        {
          label: tSeverity(severity),
          color: getSeverityColor(severity),
        },
      ])
    ),
  } satisfies ChartConfig), [t, tSeverity])
  const severityRows = SEVERITY_LEVELS.map((severity) => ({
    severity,
    label: tSeverity(severity),
    count: data?.vulnBySeverity[severity] ?? 0,
  }))
  const severityTotal = severityRows.reduce((total, item) => total + item.count, 0)
  const chartData = severityRows
    .filter((item) => item.count > 0)
    .map((item) => ({ ...item, fill: getSeverityColor(item.severity) }))
  const hasSeverityData = severityTotal > 0
  const defaultActiveIndices = chartData.reduce<number[]>((indices, item, index) => {
    if (item.severity === "critical" || item.severity === "high") {
      indices.push(index)
    }
    return indices
  }, [])
  const activeIndices = activeRiskIndex !== undefined && activeRiskIndex < chartData.length
    ? [activeRiskIndex]
    : defaultActiveIndices
  const severityDistributionLabel = severityRows
    .map((item) => `${item.label} ${item.count}`)
    .join(", ")
  const riskTrendData = useMemo(
    () => (riskHistory.data ?? []).map((item) => ({ date: item.date, totalVulns: item.totalVulns })),
    [riskHistory.data],
  )
  const riskTrendValues = riskTrendData.map((item) => item.totalVulns)
  const riskTrendMin = riskTrendValues.length > 0 ? Math.min(...riskTrendValues) : data?.totalVulns ?? 0
  const riskTrendMax = riskTrendValues.length > 0 ? Math.max(...riskTrendValues) : data?.totalVulns ?? 0
  const riskTrendPadding = Math.max(Math.ceil((riskTrendMax - riskTrendMin) * 0.18), 1)
  const riskTrendDomain: [number, number] = [
    Math.max(0, riskTrendMin - riskTrendPadding),
    riskTrendMax + riskTrendPadding,
  ]

  return (
    <OverviewSectionPanel
      title={(
        <span className="flex min-w-0 items-center gap-2">
          <RiskIcon aria-hidden="true" className="size-4 shrink-0 text-muted-foreground" />
          <span>{t("title")}</span>
        </span>
      )}
      action={(
        <Button
          size="sm"
          variant="link"
          className="h-auto self-start px-0"
          render={<Link href="/vulnerabilities/" />}
        >
          {t("viewRiskDetails")}
          <IconChevronRight aria-hidden="true" />
        </Button>
      )}
      className={cn("h-full", className)}
      contentClassName="pt-1"
    >
      {data ? (
        <div className={OVERVIEW_RISK_SUMMARY_GRID_CLASS}>
          <div className={OVERVIEW_RISK_RING_LEGEND_CLASS}>
            <div
              aria-label={`${t("severityDistribution")}: ${severityDistributionLabel}`}
              className={OVERVIEW_RISK_RING_CLASS}
              role="img"
            >
              <ChartContainer config={chartConfig} className="size-full aspect-square overview-ring-enter">
                <PieChart>
                  {hasSeverityData ? (
                    <ChartTooltip
                      cursor={false}
                      content={<ChartTooltipContent nameKey="severity" hideLabel />}
                    />
                  ) : null}
                  {hasSeverityData ? (
                    <Pie
                      activeIndex={activeIndices}
                      activeShape={renderActiveRiskSector}
                      data={chartData}
                      dataKey="count"
                      endAngle={450}
                      innerRadius={58}
                      isAnimationActive={false}
                      nameKey="severity"
                      onMouseEnter={(_, index) => setActiveRiskIndex(index)}
                      onMouseLeave={() => setActiveRiskIndex(undefined)}
                      outerRadius={78}
                      paddingAngle={2}
                      startAngle={90}
                    >
                      {chartData.map((item) => (
                        <Cell key={item.severity} fill={item.fill} />
                      ))}
                    </Pie>
                  ) : (
                    <Pie
                      data={[{ severity: "track", count: 1, fill: "var(--muted)" }]}
                      dataKey="count"
                      innerRadius={58}
                      isAnimationActive={false}
                      nameKey="severity"
                      outerRadius={78}
                    >
                      <Cell fill="var(--muted)" />
                    </Pie>
                  )}
                </PieChart>
              </ChartContainer>
              <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center rounded-full text-center">
                <span data-featured="true" className={textRole.metricValueDisplay}>
                  <OverviewNumberTicker value={data.totalVulns} locale={locale} />
                </span>
                <span className={textRole.badgeSubtle}>{t("open")}</span>
              </div>
            </div>
            <div className={OVERVIEW_RISK_LEGEND_CLASS}>
              {severityRows.map((item) => (
                <div key={item.severity} className={OVERVIEW_RISK_LEGEND_ROW_CLASS}>
                  <span className="flex min-w-0 items-center gap-2">
                    <span aria-hidden="true" className="size-2.5 shrink-0 rounded-full" style={{ backgroundColor: getSeverityColor(item.severity) }} />
                    <span className={cn("truncate", textRole.bodySubtle)}>{item.label}</span>
                  </span>
                  <span className={cn("tabular-nums", textRole.metadataValueStrong, SEVERITY_TEXT_CLASSNAMES[item.severity])}>
                    <OverviewNumberTicker value={item.count} locale={locale} />
                  </span>
                </div>
              ))}
            </div>
          </div>
          <div className={OVERVIEW_RISK_SUMMARY_METRICS_CLASS}>
            <div className="min-w-0 space-y-4">
              <div className={OVERVIEW_TREND_HEADER_CLASS}>
                <h3 className={textRole.sectionTitle}>{t("trendTitle")}</h3>
              </div>
              {riskHistory.isPending && riskTrendData.length === 0 ? (
                <Skeleton className={cn("w-full", OVERVIEW_RISK_TREND_CHART_CLASS)} />
              ) : riskTrendData.length > 0 ? (
                <div
                  aria-label={t("trendAriaLabel")}
                  className={cn("overview-chart-enter", OVERVIEW_RISK_TREND_CHART_CLASS)}
                  role="img"
                >
                  <OverviewAreaChart
                    data={riskTrendData}
                    dataKey="totalVulns"
                    gradientId="overviewRiskTrendFill"
                    margin={RISK_TREND_CHART_MARGIN}
                    tooltipFormatter={(value) => [value.toLocaleString(locale), t("open")]}
                    tooltipLabelFormatter={(label) => formatOverviewDateTick(label, locale)}
                    xAxisDataKey="date"
                    color={getTrendToneColorVar("negative")}
                    contentStyle={RISK_TREND_TOOLTIP_STYLE}
                    isAnimationActive={false}
                    showGrid={false}
                    showXAxis={false}
                    showYAxis={false}
                    yAxisDomain={riskTrendDomain}
                  />
                </div>
              ) : (
                <div className={cn("flex items-center justify-center", OVERVIEW_RISK_TREND_CHART_CLASS)}>
                  <p className={textRole.caption}>{t("trendUnavailable")}</p>
                </div>
              )}
              <div className={OVERVIEW_RISK_SUMMARY_DETAILS_CLASS}>
                <div className={OVERVIEW_RISK_SUMMARY_DETAIL_ROW_CLASS}>
                  <span className={textRole.caption}>{t("newToday")}</span>
                  <span className={cn("shrink-0 tabular-nums", textRole.bodyStrong, getTrendToneTextClass(newVulnerabilityTone))}>{formatOverviewSignedChange(newVulnerabilityCount, locale)}</span>
                </div>
                {auditStats ? (
                  <>
                    <div className={OVERVIEW_RISK_SUMMARY_DETAIL_ROW_CLASS}>
                      <span className={textRole.caption}>{t("audited")}</span>
                      <span className={cn("shrink-0 tabular-nums", textRole.metadataValueStrong)}>
                        <OverviewNumberTicker value={auditStats.reviewedCount} locale={locale} />
                      </span>
                    </div>
                    <div className={OVERVIEW_RISK_SUMMARY_DETAIL_ROW_CLASS}>
                      <span className={textRole.caption}>{t("unaudited")}</span>
                      <span className={cn("shrink-0 tabular-nums", textRole.metadataValueStrong)}>
                        <OverviewNumberTicker value={auditStats.pendingCount} locale={locale} />
                      </span>
                    </div>
                  </>
                ) : (
                  <p className={textRole.caption}>{t("auditUnavailable")}</p>
                )}
              </div>
            </div>
          </div>
        </div>
      ) : assets.error ? (
        <AppErrorState
          error={normalizeError(assets.error, { notFoundKind: "unexpected-error" })}
          title={t("loadFailed")}
          description={t("loadFailedDescription")}
          onRetry={assets.refetch}
          variant="section"
          className="min-h-48 py-4"
        />
      ) : (
        <div className="flex min-h-48 items-center justify-center text-center">
          <p className={textRole.bodySubtle}>{t("unavailable")}</p>
        </div>
      )}
    </OverviewSectionPanel>
  )
}
