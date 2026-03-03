"use client"

import { memo } from "react"
import { useTranslations } from "next-intl"
import { Card, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import {
  IconPlayerPlay,
  IconClock,
  IconCircleCheck,
  IconCircleX,
  IconBan,
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
      <CardHeader className="p-4 pb-2 @[200px]/card:p-6 @[200px]/card:pt-7 @[200px]/card:pb-2">
        <CardDescription className="flex items-center gap-1.5 @[200px]/card:gap-2 text-xs @[200px]/card:text-sm">
          {icon}
          <span className="truncate">{title}</span>
        </CardDescription>
        {loading ? (
          <Skeleton className="h-8 w-16 @[200px]/card:w-24" />
        ) : (
          <CardTitle className="text-xl font-semibold tabular-nums @[200px]/card:text-2xl @[250px]/card:text-3xl">
            {typeof value === "number" ? value.toLocaleString() : value}
          </CardTitle>
        )}
      </CardHeader>
      <CardFooter className="flex-col items-start gap-1 p-4 pt-0 @[200px]/card:p-6 @[200px]/card:pt-0">
        <div className="text-xs text-muted-foreground @[200px]/card:text-sm line-clamp-1 truncate w-full">{footer}</div>
      </CardFooter>
    </Card>
  )
})

export function ScanHistoryStatCards() {
  const { data, isLoading } = useScanStatistics()
  const t = useTranslations("scan.history.stats")
  const tCommon = useTranslations("common.status")

  return (
    <div className="grid grid-cols-3 gap-2 @2xl/main:gap-4 @xl/main:grid-cols-5 bauhaus-stats-row-5 text-[#333]">
      <StatCard
        title={tCommon("pending")}
        value={data?.pending ?? 0}
        icon={<IconClock className="size-4 text-blue-500" />}
        loading={isLoading}
        footer={t("pendingScans")}
        index={1}
      />
      <StatCard
        title={tCommon("running")}
        value={data?.running ?? 0}
        icon={<IconPlayerPlay className="size-4 text-amber-500" />}
        loading={isLoading}
        footer={t("runningScans")}
        index={2}
      />
      <StatCard
        title={tCommon("completed")}
        value={data?.completed ?? 0}
        icon={<IconCircleCheck className="size-4 text-emerald-500" />}
        loading={isLoading}
        footer={t("completedScans")}
        index={3}
      />
      <StatCard
        title={tCommon("failed")}
        value={data?.failed ?? 0}
        icon={<IconCircleX className="size-4 text-red-500" />}
        loading={isLoading}
        footer={t("failedScans")}
        index={4}
      />
      <StatCard
        title={tCommon("cancelled")}
        value={data?.cancelled ?? 0}
        icon={<IconBan className="size-4 text-slate-400" />}
        loading={isLoading}
        footer={t("cancelledScans")}
        index={5}
      />
    </div>
  )
}
