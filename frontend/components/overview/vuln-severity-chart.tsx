"use client"

import { useMemo } from "react"
import { PieChart, Pie, Cell, Label } from "recharts"
import { useAssetStatistics } from "@/hooks/use-overview"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import { Skeleton } from "@/components/ui/skeleton"
import { useTranslations } from "next-intl"
import { SEVERITY_LEVELS, getSeverityColor } from "@/lib/severity-config"
import { IconShield } from "@/components/icons"

export function VulnSeverityChart() {
  const { data, isLoading } = useAssetStatistics()
  const t = useTranslations("overview.vulnDistribution")
  const tSeverity = useTranslations("severity")

  const chartConfig = useMemo(() => ({
    count: {
      label: t("count"),
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

  const vulnData = data?.vulnBySeverity
  const allData = useMemo(
    () =>
      SEVERITY_LEVELS.map((severity) => ({
        severity,
        count: vulnData?.[severity] ?? 0,
        fill: getSeverityColor(severity),
      })),
    [vulnData]
  )
  // The pie chart only shows those with data
  const chartData = allData.filter(item => item.count > 0)

  const total = allData.reduce((sum, item) => sum + item.count, 0)

  return (
    <Card className="overview-chart-card flex h-[300px] flex-col overflow-hidden">
      <div className="overview-chart-kicker chart-kicker">
        <IconShield className="size-4" />
        <span>{t("kicker")}</span>
      </div>
      <CardHeader className="overview-chart-header">
        <CardTitle className="overview-chart-title">{t("title")}</CardTitle>
        <CardDescription>{t("description")}</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="flex h-45 items-center justify-center">
            <Skeleton className="h-30 w-30 rounded-full" />
          </div>
        ) : total === 0 ? (
          <div className="flex h-45 items-center justify-center text-muted-foreground">
            {t("noData")}
          </div>
        ) : (
          <div className="flex flex-col gap-4 items-center">
            <ChartContainer config={chartConfig} className="aspect-square h-35">
              <PieChart>
                <ChartTooltip
                  content={<ChartTooltipContent nameKey="severity" hideLabel />}
                />
                <Pie
                  data={chartData}
                  dataKey="count"
                  nameKey="severity"
                  innerRadius={45}
                  outerRadius={70}
                  paddingAngle={2}
                >
                  {chartData.map((entry) => (
                    <Cell key={entry.severity} fill={entry.fill} />
                  ))}
                  <Label
                    content={({ viewBox }) => {
                      if (viewBox && "cx" in viewBox && "cy" in viewBox) {
                        return (
                          <text
                            x={viewBox.cx}
                            y={viewBox.cy}
                            textAnchor="middle"
                            dominantBaseline="middle"
                          >
                            <tspan
                              x={viewBox.cx}
                              y={viewBox.cy}
                              className="fill-foreground font-bold text-2xl"
                            >
                              {total}
                            </tspan>
                            <tspan
                              x={viewBox.cx}
                              y={(viewBox.cy || 0) + 18}
                              className="fill-muted-foreground text-xs"
                            >
                              {t("vulns")}
                            </tspan>
                          </text>
                        )
                      }
                    }}
                  />
                </Pie>
              </PieChart>
            </ChartContainer>
            <div className="border-t flex flex-wrap gap-x-4 gap-y-1.5 justify-end mt-3 pt-3 text-sm">
              {allData.map((item) => (
                <div key={item.severity} className="flex gap-1.5 items-center">
                  <div 
                    className="h-2.5 rounded-full w-2.5" 
                    style={{ backgroundColor: item.fill }}
                  />
                  <span className={item.count > 0 ? "text-foreground" : "text-muted-foreground"}>
                    {chartConfig[item.severity as keyof typeof chartConfig]?.label}
                  </span>
                  <span className={item.count > 0 ? "font-medium" : "text-muted-foreground"}>{item.count}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
