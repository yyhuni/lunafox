"use client"

import { memo } from "react"
import { useTranslations } from "next-intl"
import { Card, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import {
  IconRadar,
  IconPlayerPlay,
  IconBug,
  IconStack2,
} from "@/components/icons"
import { useScanStatistics } from "@/hooks/use-scans"

const StatCard = memo(function StatCard({
  title,
  value,
  icon,
  footer,
  loading,
  index,
}: {
  title: string
  value: string | number
  icon: React.ReactNode
  footer: string
  loading?: boolean
  index?: number
}) {
  const formattedIndex = index !== undefined ? String(index).padStart(2, "0") : undefined

  return (
    <Card className="@container/card bauhaus-stat-card" data-card-index={formattedIndex}>
      <CardHeader className="pt-7">
        <CardDescription className="flex items-center gap-2">
          {icon}
          {title}
        </CardDescription>
        {loading ? (
          <Skeleton className="h-8 w-24" />
        ) : (
          <CardTitle className="text-2xl font-semibold tabular-nums @[250px]/card:text-3xl">
            {typeof value === "number" ? value.toLocaleString() : value}
          </CardTitle>
        )}
      </CardHeader>
      <CardFooter className="flex-col items-start gap-1.5 text-sm">
        <div className="text-muted-foreground">{footer}</div>
      </CardFooter>
    </Card>
  )
})

export function ScanHistoryStatCards() {
  const { data, isLoading } = useScanStatistics()
  const t = useTranslations("scan.history.stats")

  return (
    <div className="grid grid-cols-1 gap-4 @xl/main:grid-cols-2 @5xl/main:grid-cols-4 bauhaus-stats-row">
      <StatCard
        title={t("totalScans")}
        value={data?.total ?? 0}
        icon={<IconRadar className="size-4" />}
        loading={isLoading}
        footer={t("allScanTasks")}
        index={1}
      />
      <StatCard
        title={t("running")}
        value={data?.running ?? 0}
        icon={<IconPlayerPlay className="size-4" />}
        loading={isLoading}
        footer={t("runningScans")}
        index={2}
      />
      <StatCard
        title={t("vulnsFound")}
        value={data?.totalVulns ?? 0}
        icon={<IconBug className="size-4" />}
        loading={isLoading}
        footer={t("completedScansFound")}
        index={3}
      />
      <StatCard
        title={t("assetsFound")}
        value={data?.totalAssets ?? 0}
        icon={<IconStack2 className="size-4" />}
        loading={isLoading}
        footer={t("assetTypes")}
        index={4}
      />
    </div>
  )
}
