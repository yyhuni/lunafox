"use client"

import { useMemo } from "react"
import { Bar, BarChart, Cell, LabelList, XAxis, YAxis } from "recharts"
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
import { getAssetDistributionChartColor } from "@/lib/chart-config"
import { textRole } from "@/lib/typography"
import { useTranslations } from "next-intl"

export function AssetDistributionChart() {
  const { data, isLoading } = useAssetStatistics()
  const t = useTranslations("overview.assetDistribution")

  const chartConfig = useMemo(() => ({
    count: {
      label: t("count"),
    },
    subdomain: {
      label: t("subdomains"),
      color: getAssetDistributionChartColor("subdomain"),
    },
    ip: {
      label: t("ipAddresses"),
      color: getAssetDistributionChartColor("ip"),
    },
    endpoint: {
      label: t("endpoints"),
      color: getAssetDistributionChartColor("endpoint"),
    },
    website: {
      label: t("websites"),
      color: getAssetDistributionChartColor("website"),
    },
  } satisfies ChartConfig), [t])

  const chartData = useMemo(() => [
    { key: "subdomain", name: t("subdomains"), count: data?.totalSubdomains ?? 0, fill: getAssetDistributionChartColor("subdomain") },
    { key: "ip", name: t("ipAddresses"), count: data?.totalIps ?? 0, fill: getAssetDistributionChartColor("ip") },
    { key: "endpoint", name: t("endpoints"), count: data?.totalEndpoints ?? 0, fill: getAssetDistributionChartColor("endpoint") },
    { key: "website", name: t("websites"), count: data?.totalWebsites ?? 0, fill: getAssetDistributionChartColor("website") },
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
          <ChartContainer config={chartConfig} className="aspect-auto h-40 w-full">
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
                {chartData.map((entry) => (
                  <Cell key={entry.key} fill={entry.fill} />
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
          <div className="border-t flex gap-1.5 items-center justify-end mt-3 pt-3 text-sm">
            <span className="text-muted-foreground">{t("totalAssets")}:</span>
            <span className={textRole.metadataValueStrong}>{total}</span>
          </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}
