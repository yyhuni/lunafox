"use client"

import { memo, useEffect } from "react"
import { useAssetStatistics } from "@/hooks/use-overview"
import { Card, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { IconPlayerPlay, IconTrendingUp, IconTrendingDown, semanticIcons } from "@/components/icons"
import { useTranslations } from "next-intl"
import { motion, useSpring, useTransform } from "framer-motion"
import { getTrendToneBadgeClass } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

const VulnerabilitiesMetricIcon = semanticIcons.concept.vulnerability
const AssetMetricIcon = semanticIcons.concept.asset
const TargetMetricIcon = semanticIcons.concept.target

function NumberTicker({ value }: { value: number }) {
  // Initialize from 0 to create a "count up" effect on mount
  const spring = useSpring(0, { mass: 0.8, stiffness: 75, damping: 15 })
  const display = useTransform(spring, (current) => Math.round(current).toLocaleString())

  useEffect(() => {
    spring.set(value)
  }, [value, spring])

  return <motion.span>{display}</motion.span>
}

const TrendBadge = memo(function TrendBadge({ change }: { change: number }) {
  if (change === 0) return null
  
  const isPositive = change > 0
  return (
    <Badge 
      variant="outline" 
      className={getTrendToneBadgeClass(isPositive ? "positive" : "negative")}
    >
      {isPositive ? <IconTrendingUp className="mr-1 size-3" /> : <IconTrendingDown className="mr-1 size-3" />}
      {isPositive ? '+' : ''}{change}
    </Badge>
  )
})

function MetricSparkline() {
  return (
    <svg aria-hidden="true" className="h-6 w-24 text-muted-foreground/20" viewBox="0 0 112 32">
      <path
        d="M2 24 L14 19 L24 22 L34 14 L45 17 L55 8 L66 15 L76 10 L86 13 L98 7 L110 10"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.5"
      />
    </svg>
  )
}

const StatCard = memo(function StatCard({
  title,
  value,
  change,
  icon,
  footer,
  loading,
  index,
}: {
  title: string
  value: string | number
  change?: number
  icon: React.ReactNode
  footer: string
  loading?: boolean
  index?: number
}) {
  // Format serial number as two digits (01, 02, 03, 04)
  const formattedIndex = index !== undefined ? String(index).padStart(2, "0") : undefined
  
  return (
    <Card 
      className="@container/card min-h-28 justify-between border-t border-border/70 py-4 shadow-none" 
      data-card-index={formattedIndex}
    >
      <CardHeader className="grid-cols-[1fr_auto] gap-2 px-6">
        <CardDescription className={cn("flex items-center gap-2", textRole.metadataLabel)}>
          <span className="text-muted-foreground">{icon}</span>
          <span>{title}</span>
        </CardDescription>
        {loading ? (
          <Skeleton className="h-5 w-28" />
        ) : (
          <CardTitle className={cn("row-start-2 flex items-center gap-3", textRole.metricValueDisplay)}>
            <span>{typeof value === "number" ? <NumberTicker value={value} /> : value}</span>
            {!loading && change !== undefined ? <TrendBadge change={change} /> : null}
          </CardTitle>
        )}
        <div className="col-start-2 row-span-2 row-start-1 self-end opacity-70">
          <MetricSparkline />
        </div>
      </CardHeader>
      <div className={cn("px-6", textRole.bodySubtle)}>{footer}</div>
    </Card>
  )
})

export function OverviewStatCards({ className, gridClassName }: { className?: string; gridClassName?: string } = {}) {
  const { data, isLoading } = useAssetStatistics()
  const t = useTranslations("overview.statCards")

  return (
    <div className={cn("px-4 lg:px-6", className)}>
      <div className={cn("overview-stats-row grid grid-cols-1 gap-4 @xl/main:grid-cols-2 @5xl/main:grid-cols-4", gridClassName)}>
        <StatCard
          title={t("assetsFound")}
          value={data?.totalAssets ?? 0}
          change={data?.changeAssets}
          icon={<AssetMetricIcon className="size-4" />}
          loading={isLoading}
          footer={t("assetsFooter")}
          index={1}
        />
        <StatCard
          title={t("vulnsFound")}
          value={data?.totalVulns ?? 0}
          change={data?.changeVulns}
          icon={<VulnerabilitiesMetricIcon className="size-4" />}
          loading={isLoading}
          footer={t("vulnsFooter")}
          index={2}
        />
        <StatCard
          title={t("monitoredTargets")}
          value={data?.totalTargets ?? 0}
          change={data?.changeTargets}
          icon={<TargetMetricIcon className="size-4" />}
          loading={isLoading}
          footer={t("targetsFooter")}
          index={3}
        />
        <StatCard
          title={t("runningScans")}
          value={data?.runningScans ?? 0}
          icon={<IconPlayerPlay className="size-4" />}
          loading={isLoading}
          footer={t("scansFooter")}
          index={4}
        />
      </div>
    </div>
  )
}
