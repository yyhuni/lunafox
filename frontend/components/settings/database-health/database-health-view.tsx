"use client"

import { useMemo, useCallback, type ReactNode } from "react"
import { useLocale, useTranslations } from "next-intl"
import { IconDatabase, IconServer, IconTrendingUp, IconUsers } from "@/components/icons"

import { useDatabaseHealth } from "@/hooks/use-database-health"
import type { DatabaseAlertSeverity, DatabaseHealthStatus } from "@/types/database-health.types"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { Skeleton } from "@/components/ui/skeleton"
import { Status, StatusIndicator, StatusLabel } from "@/components/ui/shadcn-io/status"
import { cn } from "@/lib/utils"
import { PageHeader } from "@/components/common/page-header"

const severityStyles: Record<DatabaseAlertSeverity, string> = {
  info: "text-[var(--success)] border-[var(--success)]/20 bg-[var(--success)]/10",
  warning: "text-[var(--warning)] border-[var(--warning)]/20 bg-[var(--warning)]/10",
  critical: "text-[var(--error)] border-[var(--error)]/20 bg-[var(--error)]/10",
}

function statusFromThreshold(
  value: number,
  warn: number,
  critical: number,
  higherIsWorse: boolean = true
): DatabaseHealthStatus {
  if (higherIsWorse) {
    if (value >= critical) return "offline"
    if (value >= warn) return "degraded"
    return "online"
  }
  if (value <= critical) return "offline"
  if (value <= warn) return "degraded"
  return "online"
}

function MetricCard({
  title,
  value,
  status,
  progress,
  icon,
  loading,
  hint,
}: {
  title: string
  value: string
  status: DatabaseHealthStatus
  progress?: number
  icon: ReactNode
  loading?: boolean
  hint?: string
}) {
  const t = useTranslations("databaseHealth")

  return (
    <Card className="@container/card">
      <CardHeader className="pb-3">
        <CardDescription className="flex items-center justify-between gap-2">
          <span className="flex items-center gap-2">
            {icon}
            {title}
          </span>
          <Status status={status} className="px-1.5 py-0.5 text-[10px]">
            <StatusIndicator />
            <StatusLabel className="text-[10px]">{t(`status.${status}`)}</StatusLabel>
          </Status>
        </CardDescription>
        {loading ? (
          <Skeleton className="h-7 w-24" />
        ) : (
          <CardTitle className="text-2xl font-semibold tabular-nums @[250px]/card:text-3xl">
            {value}
          </CardTitle>
        )}
      </CardHeader>
      {!loading && (progress !== undefined || hint) && (
        <CardContent className="pt-0">
          {progress !== undefined && (
            <Progress value={progress} className="h-2" />
          )}
          {hint && (
            <p className={cn("text-xs text-muted-foreground", progress !== undefined && "mt-2")}>{hint}</p>
          )}
        </CardContent>
      )}
    </Card>
  )
}

function SummaryItem({ label, value, loading }: { label: string; value: ReactNode; loading?: boolean }) {
  return (
    <div className="space-y-1">
      <p className="text-xs text-muted-foreground">{label}</p>
      {loading ? <Skeleton className="h-5 w-24" /> : <div className="text-sm font-medium">{value}</div>}
    </div>
  )
}

export function DatabaseHealthView() {
  const t = useTranslations("databaseHealth")
  const tPage = useTranslations("pages.databaseHealth")
  const locale = useLocale()
  const { data, isLoading } = useDatabaseHealth()
  const loading = isLoading && !data

  const formatNumber = useCallback((value: number, digits: number = 0) =>
    new Intl.NumberFormat(locale, {
      maximumFractionDigits: digits,
      minimumFractionDigits: digits,
    }).format(value), [locale])

  const signals = data?.signals
  const connectionsPercent = signals && signals.connectionsMax > 0
    ? Math.min(100, (signals.connectionsUsed / signals.connectionsMax) * 100)
    : 0
  const diskPercent = Math.min(100, signals?.diskUsagePercent ?? 0)

  const fallbackStatus: DatabaseHealthStatus = data?.status ?? "offline"
  const statusLatency = signals ? statusFromThreshold(signals.latencyP95Ms, 50, 150) : fallbackStatus
  const statusConnections = signals ? statusFromThreshold(connectionsPercent, 70, 85) : fallbackStatus
  const statusReplication = signals ? statusFromThreshold(signals.replicationLagMs, 1000, 5000) : fallbackStatus
  const statusCache = signals ? statusFromThreshold(signals.cacheHitRate, 99, 97, false) : fallbackStatus
  const statusDisk = signals ? statusFromThreshold(diskPercent, 70, 85) : fallbackStatus
  const statusDeadlocks = signals ? statusFromThreshold(signals.deadlocks1h, 1, 5) : fallbackStatus
  const statusLongTx = signals ? statusFromThreshold(signals.longTransactions, 1, 5) : fallbackStatus
  const statusBackup = signals ? statusFromThreshold(signals.backupFreshnessMinutes, 120, 360) : fallbackStatus

  const extraSignals = useMemo(
    () => [
      {
        label: t("metrics.qps"),
        value: signals ? formatNumber(signals.qps) : "--",
        status: "online" as DatabaseHealthStatus,
      },
      {
        label: t("metrics.diskUsage"),
        value: signals ? `${formatNumber(diskPercent, 1)}%` : "--",
        status: statusDisk,
        progress: diskPercent,
      },
      {
        label: t("metrics.walGenerated"),
        value: signals ? `${formatNumber(signals.walGeneratedMb24h)} MB` : "--",
        status: "online" as DatabaseHealthStatus,
      },
      {
        label: t("metrics.deadlocks"),
        value: signals ? formatNumber(signals.deadlocks1h) : "--",
        status: statusDeadlocks,
      },
      {
        label: t("metrics.longTransactions"),
        value: signals ? formatNumber(signals.longTransactions) : "--",
        status: statusLongTx,
      },
      {
        label: t("metrics.backupFreshness"),
        value: signals ? `${formatNumber(signals.backupFreshnessMinutes)} min` : "--",
        status: statusBackup,
      },
    ],
    [
      t,
      signals,
      formatNumber,
      diskPercent,
      statusDisk,
      statusDeadlocks,
      statusLongTx,
      statusBackup,
    ]
  )

  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader
        code="DBH-01"
        title={tPage("title")}
        description={tPage("description")}
      />

      <div className="px-4 lg:px-6 space-y-6">
        <div className="flex items-center justify-end">
          {loading ? (
            <Skeleton className="h-7 w-28" />
          ) : (
            <Status status={data?.status ?? "offline"} className="px-2 py-1">
              <StatusIndicator />
              <StatusLabel>{t(`status.${data?.status ?? "offline"}`)}</StatusLabel>
            </Status>
          )}
        </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("overview.title")}</CardTitle>
          <CardDescription>{t("overview.description")}</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <SummaryItem
              label={t("overview.role")}
              value={
                <Badge variant="outline">
                  {t(`overview.roleValue.${data?.role ?? "primary"}`)}
                </Badge>
              }
              loading={loading}
            />
            <SummaryItem label={t("overview.region")} value={data?.region ?? "--"} loading={loading} />
            <SummaryItem label={t("overview.version")} value={data?.version ?? "--"} loading={loading} />
            <SummaryItem
              label={t("overview.readOnly")}
              value={
                <Badge variant="outline" className={data?.readOnly ? "text-[var(--warning)]" : undefined}>
                  {data?.readOnly ? t("overview.readOnlyYes") : t("overview.readOnlyNo")}
                </Badge>
              }
              loading={loading}
            />
            <SummaryItem label={t("overview.uptime")} value={data?.uptime ?? "--"} loading={loading} />
            <SummaryItem label={t("overview.lastCheck")} value={data?.lastCheckAt ?? "--"} loading={loading} />
          </div>
        </CardContent>
      </Card>

      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("sections.core")}
          </h2>
        </div>
        <div className="grid grid-cols-1 gap-4 @xl/main:grid-cols-2 @4xl/main:grid-cols-4">
          <MetricCard
            title={t("metrics.latencyP95")}
            value={signals ? `${formatNumber(signals.latencyP95Ms)} ms` : "--"}
            status={statusLatency}
            icon={<IconTrendingUp className="size-4" />}
            loading={loading}
            hint={t("hints.latencyP95")}
          />
          <MetricCard
            title={t("metrics.connections")}
            value={
              signals ? `${formatNumber(signals.connectionsUsed)} / ${formatNumber(signals.connectionsMax)}` : "--"
            }
            status={statusConnections}
            icon={<IconUsers className="size-4" />}
            loading={loading}
            progress={connectionsPercent}
            hint={t("hints.connections", { value: formatNumber(connectionsPercent, 1) })}
          />
          <MetricCard
            title={t("metrics.replicationLag")}
            value={signals ? `${formatNumber(signals.replicationLagMs)} ms` : "--"}
            status={statusReplication}
            icon={<IconServer className="size-4" />}
            loading={loading}
            hint={t("hints.replicationLag")}
          />
          <MetricCard
            title={t("metrics.cacheHitRate")}
            value={signals ? `${formatNumber(signals.cacheHitRate, 1)}%` : "--"}
            status={statusCache}
            icon={<IconDatabase className="size-4" />}
            loading={loading}
            progress={signals?.cacheHitRate ?? 0}
            hint={t("hints.cacheHitRate")}
          />
        </div>
      </div>

      <div className="grid gap-4 @xl/main:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{t("sections.signals")}</CardTitle>
            <CardDescription>{t("sections.signalsDesc")}</CardDescription>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="space-y-3">
                <Skeleton className="h-6 w-full" />
                <Skeleton className="h-6 w-full" />
                <Skeleton className="h-6 w-full" />
              </div>
            ) : (
              <div className="space-y-3">
                {extraSignals.map((signal) => (
                  <div key={signal.label} className="flex items-center justify-between gap-4">
                    <div className="text-sm text-muted-foreground">{signal.label}</div>
                    <div className="flex items-center gap-3">
                      <span className="text-sm font-medium tabular-nums">{signal.value}</span>
                      <Status status={signal.status} className="px-1.5 py-0.5 text-[10px]">
                        <StatusIndicator />
                        <StatusLabel className="text-[10px]">{t(`status.${signal.status}`)}</StatusLabel>
                      </Status>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("alerts.title")}</CardTitle>
            <CardDescription>{t("alerts.description")}</CardDescription>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="space-y-3">
                <Skeleton className="h-6 w-full" />
                <Skeleton className="h-6 w-full" />
                <Skeleton className="h-6 w-full" />
              </div>
            ) : data?.alerts?.length ? (
              <div className="space-y-3">
                {data.alerts.map((alert) => (
                  <div key={alert.id} className="flex items-start justify-between gap-4 border-b pb-3 last:border-b-0 last:pb-0">
                    <div>
                      <div className="flex items-center gap-2">
                        <Badge variant="outline" className={severityStyles[alert.severity]}>
                          {t(`alerts.severity.${alert.severity}`)}
                        </Badge>
                        <span className="text-sm font-medium">{alert.title}</span>
                      </div>
                      <p className="text-xs text-muted-foreground mt-1">{alert.description}</p>
                    </div>
                    <span className="text-xs text-muted-foreground">{alert.time}</span>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-sm text-muted-foreground">{t("alerts.empty")}</div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  </div>
  )
}
