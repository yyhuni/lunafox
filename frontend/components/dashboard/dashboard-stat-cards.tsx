"use client"

import { memo, useEffect } from "react"
import { useAssetStatistics } from "@/hooks/use-dashboard"
import { Card, CardAction, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { IconTarget, IconStack2, IconBug, IconPlayerPlay, IconTrendingUp, IconTrendingDown } from "@/components/icons"
import { useTranslations } from "next-intl"
import { useLocale } from "next-intl"
import { motion, useSpring, useTransform } from "framer-motion"

function NumberTicker({ value }: { value: number }) {
  // Initialize from 0 to create a "count up" effect on mount
  const spring = useSpring(0, { mass: 0.8, stiffness: 75, damping: 15 });
  const display = useTransform(spring, (current) => Math.round(current).toLocaleString());

  useEffect(() => {
    spring.set(value);
  }, [value, spring]);

  return <motion.span>{display}</motion.span>;
}

const TrendBadge = memo(function TrendBadge({ change }: { change: number }) {
  if (change === 0) return null
  
  const isPositive = change > 0
  return (
    <Badge 
      variant="outline" 
      className={isPositive 
        ? "text-[var(--success)] border-[var(--success)]/20 bg-[var(--success)]/10" 
        : "text-[var(--error)] border-[var(--error)]/20 bg-[var(--error)]/10"
      }
    >
      {isPositive ? <IconTrendingUp className="size-3 mr-1" /> : <IconTrendingDown className="size-3 mr-1" />}
      {isPositive ? '+' : ''}{change}
    </Badge>
  )
})

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
  const formattedIndex = index !== undefined ? String(index).padStart(2, '0') : undefined
  
  return (
    <Card 
      className="@container/card bauhaus-stat-card" 
      data-card-index={formattedIndex}
    >
      <CardHeader className="pt-7">
        <CardDescription className="flex items-center gap-2">
          {icon}
          {title}
        </CardDescription>
        {loading ? (
          <Skeleton className="h-8 w-24" />
        ) : (
          <CardTitle className="text-2xl font-semibold tabular-nums @[250px]/card:text-3xl">
            {typeof value === 'number' ? <NumberTicker value={value} /> : value}
          </CardTitle>
        )}
        {!loading && change !== undefined && (
          <CardAction>
            <TrendBadge change={change} />
          </CardAction>
        )}
      </CardHeader>
      <CardFooter className="flex-col items-start gap-1.5 text-sm">
        <div className="text-muted-foreground">{footer}</div>
      </CardFooter>
    </Card>
  )
})

function formatUpdateTime(dateStr: string | null, locale: string, noDataText: string) {
  if (!dateStr) return noDataText
  const date = new Date(dateStr)
  return date.toLocaleString(locale === 'zh' ? 'zh-CN' : 'en-US', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function DashboardStatCards() {
  const { data, isLoading } = useAssetStatistics()
  const t = useTranslations("dashboard.statCards")
  const locale = useLocale()

  return (
    <div className="flex flex-col gap-2 px-4 lg:px-6">
      {/* The Bauhaus theme uses a 1x4 horizontal layout, and other themes use a responsive grid. */}
      <div className="grid grid-cols-1 gap-4 @xl/main:grid-cols-2 @5xl/main:grid-cols-4 bauhaus-stats-row">
        <StatCard
          title={t("assetsFound")}
          value={data?.totalAssets ?? 0}
          change={data?.changeAssets}
          icon={<IconStack2 className="size-4" />}
          loading={isLoading}
          footer={t("assetsFooter")}
          index={1}
        />
        <StatCard
          title={t("vulnsFound")}
          value={data?.totalVulns ?? 0}
          change={data?.changeVulns}
          icon={<IconBug className="size-4" />}
          loading={isLoading}
          footer={t("vulnsFooter")}
          index={2}
        />
        <StatCard
          title={t("monitoredTargets")}
          value={data?.totalTargets ?? 0}
          change={data?.changeTargets}
          icon={<IconTarget className="size-4" />}
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
      <div className="flex items-center gap-3 mt-1 -mb-2 text-xs text-muted-foreground">
        <div className="flex-1 border-t" />
        <span>{t("updatedAt", { time: formatUpdateTime(data?.updatedAt ?? null, locale, t("noData")) })}</span>
      </div>
    </div>
  )
}
