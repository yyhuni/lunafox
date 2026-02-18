"use client"

import { useMemo } from "react"
import { Bar, BarChart, Cell, LabelList, XAxis, YAxis } from "recharts"
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

// Use CSS variables, follow theme changes
const COLORS = {
  subdomain: "var(--chart-1)",
  ip: "var(--chart-2)",
  endpoint: "var(--chart-3)",
  website: "var(--chart-4)",
}

export function AssetDistributionChart() {
  const { data, isLoading } = useAssetStatistics()
  const t = useTranslations("dashboard.assetDistribution")

  const chartConfig = useMemo(() => ({
    count: {
      label: t("count"),
    },
    subdomain: {
      label: t("subdomains"),
      color: COLORS.subdomain,
    },
    ip: {
      label: t("ipAddresses"),
      color: COLORS.ip,
    },
    endpoint: {
      label: t("endpoints"),
      color: COLORS.endpoint,
    },
    website: {
      label: t("websites"),
      color: COLORS.website,
    },
  } satisfies ChartConfig), [t])

  const chartData = useMemo(() => [
    { name: t("subdomains"), count: data?.totalSubdomains ?? 0, fill: COLORS.subdomain },
    { name: t("ipAddresses"), count: data?.totalIps ?? 0, fill: COLORS.ip },
    { name: t("endpoints"), count: data?.totalEndpoints ?? 0, fill: COLORS.endpoint },
    { name: t("websites"), count: data?.totalWebsites ?? 0, fill: COLORS.website },
  ], [data, t])

  const total = chartData.reduce((sum, item) => sum + item.count, 0)

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("title")}</CardTitle>
        <CardDescription>{t("description")}</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-4">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-4/5" />
            <Skeleton className="h-8 w-3/5" />
            <Skeleton className="h-8 w-2/5" />
          </div>
        ) : (
          <>
          <ChartContainer config={chartConfig} className="aspect-auto h-[160px] w-full">
            <BarChart
              accessibilityLayer
              data={chartData}
              layout="vertical"
              margin={{ left: 0, right: 30 }}
            >
              <YAxis
                dataKey="name"
                type="category"
                tickLine={false}
                tickMargin={10}
                axisLine={false}
                width={50}
              />
              <XAxis dataKey="count" type="number" hide />
              <ChartTooltip
                cursor={false}
                content={<ChartTooltipContent hideLabel />}
              />
              <Bar
                dataKey="count"
                layout="vertical"
                radius={4}
              >
                {chartData.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={entry.fill} />
                ))}
                <LabelList
                  dataKey="count"
                  position="right"
                  offset={8}
                  className="fill-foreground"
                  fontSize={12}
                />
              </Bar>
            </BarChart>
          </ChartContainer>
          <div className="mt-3 pt-3 border-t flex items-center justify-end gap-1.5 text-sm">
            <span className="text-muted-foreground">{t("totalAssets")}:</span>
            <span className="font-semibold">{total}</span>
          </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}
