"use client"

import { useMemo, useState } from "react"
import { Area, CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts"
import { useTranslations } from "next-intl"
import { PageHeader } from "@/components/common/page-header"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { ChartConfig, ChartContainer } from "@/components/ui/chart"
import { useStatisticsHistory } from "@/hooks/use-dashboard"
import type { StatisticsHistoryItem } from "@/types/dashboard.types"

type SeriesKey = "totalSubdomains" | "totalIps" | "totalEndpoints" | "totalWebsites"
type TrendVariant = "layered" | "mono" | "band" | "focus" | "grid"

const VARIANTS: Array<{ value: TrendVariant; label: string; desc: string }> = [
  { value: "layered", label: "叠层折线", desc: "多线并行，线宽随 hover 强调" },
  { value: "mono", label: "单线主脉冲", desc: "仅显示总资产主线" },
  { value: "band", label: "波段趋势", desc: "总资产面积 + 线形轮廓" },
  { value: "focus", label: "聚焦模式", desc: "悬停高亮，其他线降透明" },
  { value: "grid", label: "网格脉冲", desc: "工业网格背景增强秩序感" },
]

function fillMissingDates(data: StatisticsHistoryItem[] | undefined, days: number): StatisticsHistoryItem[] {
  if (!data || data.length === 0) return []
  const dataMap = new Map(data.map(item => [item.date, item]))
  const earliestDate = new Date(data[0].date)
  const result: StatisticsHistoryItem[] = []
  const startDate = new Date(earliestDate)
  startDate.setDate(startDate.getDate() - (days - data.length))

  for (let i = 0; i < days; i++) {
    const currentDate = new Date(startDate)
    currentDate.setDate(startDate.getDate() + i)
    const dateStr = currentDate.toISOString().split("T")[0]
    const existing = dataMap.get(dateStr)
    result.push(existing ?? {
      date: dateStr,
      totalTargets: 0,
      totalSubdomains: 0,
      totalIps: 0,
      totalEndpoints: 0,
      totalWebsites: 0,
      totalVulns: 0,
      totalAssets: 0,
    })
  }

  return result
}

function formatDate(dateStr: string) {
  const date = new Date(dateStr)
  return `${date.getMonth() + 1}/${date.getDate()}`
}

function formatNumber(value: number) {
  if (value >= 1000000) return `${(value / 1000000).toFixed(1)}M`
  if (value >= 1000) return `${(value / 1000).toFixed(1)}K`
  return value.toString()
}

function LineVariantCard({
  variant,
  data,
  chartConfig,
  title,
  description,
  totals,
}: {
  variant: TrendVariant
  data: StatisticsHistoryItem[]
  chartConfig: ChartConfig
  title: string
  description: string
  totals: StatisticsHistoryItem | null
}) {
  const [hoveredLine, setHoveredLine] = useState<SeriesKey | null>(null)
  const showTotalOnly = variant === "mono" || variant === "band"
  const showGrid = variant === "grid"

  return (
    <Card className="overflow-hidden">
      <CardHeader className="space-y-1">
        <CardTitle className="text-sm">{title}</CardTitle>
        <p className="text-xs text-muted-foreground">{description}</p>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className={showGrid ? "rounded-md border border-border/60 bg-muted/10 p-2" : undefined}>
          <ChartContainer config={chartConfig} className="aspect-auto h-[160px] w-full">
            <LineChart
              accessibilityLayer
              data={data}
              margin={{ left: 0, right: 12, top: 12, bottom: 0 }}
              onMouseLeave={() => setHoveredLine(null)}
            >
              <CartesianGrid vertical={showGrid} strokeDasharray="3 3" />
              <XAxis
                dataKey="date"
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                tickFormatter={formatDate}
                fontSize={12}
              />
              <YAxis
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                width={45}
                fontSize={12}
                tickFormatter={formatNumber}
              />
              {variant === "band" && (
                <Area
                  type="monotone"
                  dataKey="totalAssets"
                  stroke="var(--color-totalAssets)"
                  fill="var(--color-totalAssets)"
                  fillOpacity={0.18}
                  strokeWidth={2.5}
                  dot={{ r: 3, fill: "var(--color-totalAssets)" }}
                />
              )}
              {showTotalOnly && variant !== "band" && (
                <Line
                  dataKey="totalAssets"
                  type="monotone"
                  stroke="var(--color-totalAssets)"
                  strokeWidth={2.5}
                  dot={{ r: 3, fill: "var(--color-totalAssets)" }}
                />
              )}
              {!showTotalOnly && (
                <>
                  <Line
                    dataKey="totalSubdomains"
                    type="monotone"
                    stroke="var(--color-totalSubdomains)"
                    strokeWidth={hoveredLine === "totalSubdomains" ? 4 : 2}
                    strokeOpacity={variant === "focus" && hoveredLine && hoveredLine !== "totalSubdomains" ? 0.18 : 1}
                    dot={{ r: 3, fill: "var(--color-totalSubdomains)" }}
                    style={{ cursor: "pointer", transition: "stroke-width 0.15s" }}
                    onMouseEnter={() => setHoveredLine("totalSubdomains")}
                  />
                  <Line
                    dataKey="totalIps"
                    type="monotone"
                    stroke="var(--color-totalIps)"
                    strokeWidth={hoveredLine === "totalIps" ? 4 : 2}
                    strokeOpacity={variant === "focus" && hoveredLine && hoveredLine !== "totalIps" ? 0.18 : 1}
                    dot={{ r: 3, fill: "var(--color-totalIps)" }}
                    style={{ cursor: "pointer", transition: "stroke-width 0.15s" }}
                    onMouseEnter={() => setHoveredLine("totalIps")}
                  />
                  <Line
                    dataKey="totalEndpoints"
                    type="monotone"
                    stroke="var(--color-totalEndpoints)"
                    strokeWidth={hoveredLine === "totalEndpoints" ? 4 : 2}
                    strokeOpacity={variant === "focus" && hoveredLine && hoveredLine !== "totalEndpoints" ? 0.18 : 1}
                    dot={{ r: 3, fill: "var(--color-totalEndpoints)" }}
                    style={{ cursor: "pointer", transition: "stroke-width 0.15s" }}
                    onMouseEnter={() => setHoveredLine("totalEndpoints")}
                  />
                  <Line
                    dataKey="totalWebsites"
                    type="monotone"
                    stroke="var(--color-totalWebsites)"
                    strokeWidth={hoveredLine === "totalWebsites" ? 4 : 2}
                    strokeOpacity={variant === "focus" && hoveredLine && hoveredLine !== "totalWebsites" ? 0.18 : 1}
                    dot={{ r: 3, fill: "var(--color-totalWebsites)" }}
                    style={{ cursor: "pointer", transition: "stroke-width 0.15s" }}
                    onMouseEnter={() => setHoveredLine("totalWebsites")}
                  />
                </>
              )}
            </LineChart>
          </ChartContainer>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
          <span>{totals ? formatDate(totals.date) : "—"}</span>
          {showTotalOnly ? (
            <span className="flex items-center gap-2">
              <span className="h-2.5 w-2.5 rounded-full bg-primary/70" />
              <span>总资产</span>
              <span className="font-medium text-foreground">{totals?.totalAssets ?? 0}</span>
            </span>
          ) : (
            <span className="flex items-center gap-2">
              <span className="h-2.5 w-2.5 rounded-full bg-blue-500" />
              <span className="font-medium text-foreground">{totals?.totalSubdomains ?? 0}</span>
              <span className="h-2.5 w-2.5 rounded-full bg-emerald-500" />
              <span className="font-medium text-foreground">{totals?.totalWebsites ?? 0}</span>
              <span className="h-2.5 w-2.5 rounded-full bg-orange-500" />
              <span className="font-medium text-foreground">{totals?.totalIps ?? 0}</span>
            </span>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

export default function AssetPulseLineDemosPage() {
  const t = useTranslations("dashboard.assetTrend")
  const tDist = useTranslations("dashboard.assetDistribution")
  const { data: rawData } = useStatisticsHistory(7)

  const chartConfig = useMemo(() => ({
    totalAssets: {
      label: tDist("totalAssets"),
      color: "var(--primary)",
    },
    totalSubdomains: {
      label: t("subdomains"),
      color: "#3b82f6",
    },
    totalIps: {
      label: t("ips"),
      color: "#f97316",
    },
    totalEndpoints: {
      label: t("endpoints"),
      color: "#eab308",
    },
    totalWebsites: {
      label: t("websites"),
      color: "#22c55e",
    },
  } satisfies ChartConfig), [t, tDist])

  const data = useMemo(() => fillMissingDates(rawData, 7), [rawData])
  const latest = useMemo(
    () => (data.length > 0 ? data[data.length - 1] : null),
    [data]
  )

  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader
        code="DASH-PULSE"
        title="Asset Pulse · Line Variants"
        description="折线图为核心的 ASSET PULSE 方案（5 套）"
      />
      <div className="grid gap-6 px-4 lg:px-6 md:grid-cols-2">
        {VARIANTS.map((item, index) => (
          <LineVariantCard
            key={item.value}
            variant={item.value}
            data={data}
            chartConfig={chartConfig}
            title={`${index + 1}. ${item.label}`}
            description={item.desc}
            totals={latest}
          />
        ))}
      </div>
    </div>
  )
}
