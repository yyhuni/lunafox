"use client"

import { useMemo } from "react"
import { PieChart, Pie, Cell, Label } from "recharts"
import { useAssetStatistics } from "@/hooks/use-dashboard"
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
import { SEVERITY_COLORS } from "@/lib/severity-config"
import { IconShield } from "@/components/icons"

export function VulnSeverityChart() {
  const { data, isLoading } = useAssetStatistics()
  const t = useTranslations("dashboard.vulnDistribution")
  const tSeverity = useTranslations("severity")

  const chartConfig = useMemo(() => ({
    count: {
      label: "Count",
    },
    critical: {
      label: tSeverity("critical"),
      color: SEVERITY_COLORS.critical,
    },
    high: {
      label: tSeverity("high"),
      color: SEVERITY_COLORS.high,
    },
    medium: {
      label: tSeverity("medium"),
      color: SEVERITY_COLORS.medium,
    },
    low: {
      label: tSeverity("low"),
      color: SEVERITY_COLORS.low,
    },
    info: {
      label: tSeverity("info"),
      color: SEVERITY_COLORS.info,
    },
  } satisfies ChartConfig), [tSeverity])

  const vulnData = data?.vulnBySeverity
  const allData = useMemo(() => [
    { severity: "critical", count: vulnData?.critical ?? 0, fill: SEVERITY_COLORS.critical },
    { severity: "high", count: vulnData?.high ?? 0, fill: SEVERITY_COLORS.high },
    { severity: "medium", count: vulnData?.medium ?? 0, fill: SEVERITY_COLORS.medium },
    { severity: "low", count: vulnData?.low ?? 0, fill: SEVERITY_COLORS.low },
    { severity: "info", count: vulnData?.info ?? 0, fill: SEVERITY_COLORS.info },
  ], [vulnData])
  // The pie chart only shows those with data
  const chartData = allData.filter(item => item.count > 0)

  const total = allData.reduce((sum, item) => sum + item.count, 0)

  return (
    <Card className="flex flex-col h-[300px] overflow-hidden [[data-theme=bauhaus]_&]:pt-0 [[data-theme=bauhaus]_&]:gap-0">
      {/* Bauhaus Style Kicker Title */}
      <div className="bauhaus-kicker hidden [[data-theme=bauhaus]_&]:flex">
        <IconShield className="size-4" />
        <span>SEVERITY MIX</span>
      </div>
      <CardHeader className="[[data-theme=bauhaus]_&]:pt-4">
        <CardTitle className="[[data-theme=bauhaus]_&]:hidden">{t("title")}</CardTitle>
        <CardDescription>{t("description")}</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="flex items-center justify-center h-[180px]">
            <Skeleton className="h-[120px] w-[120px] rounded-full" />
          </div>
        ) : total === 0 ? (
          <div className="flex items-center justify-center h-[180px] text-muted-foreground">
            {t("noData")}
          </div>
        ) : (
          <div className="flex flex-col items-center gap-4">
            <ChartContainer config={chartConfig} className="aspect-square h-[140px]">
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
                              className="fill-foreground text-2xl font-bold"
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
            <div className="mt-3 pt-3 border-t flex flex-wrap justify-end gap-x-4 gap-y-1.5 text-sm">
              {allData.map((item) => (
                <div key={item.severity} className="flex items-center gap-1.5">
                  <div 
                    className="h-2.5 w-2.5 rounded-full" 
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
