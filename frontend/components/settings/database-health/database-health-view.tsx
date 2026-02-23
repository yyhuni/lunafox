"use client"

import { useCallback, useMemo, type ReactNode } from "react"
import { useLocale, useTranslations } from "next-intl"
import { IconAlertTriangle, IconClock, IconDatabase, IconTrendingUp, IconUsers } from "@/components/icons"

import { useDatabaseHealth } from "@/hooks/use-database-health"
import type {
  DatabaseAlertSeverity,
  DatabaseHealthStatus,
  DatabaseUnavailableReason,
} from "@/types/database-health.types"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { Skeleton } from "@/components/ui/skeleton"
import { Status, StatusIndicator, StatusLabel } from "@/components/ui/shadcn-io/status"
import { cn } from "@/lib/utils"
import { PageHeader } from "@/components/common/page-header"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"

const severityStyles: Record<DatabaseAlertSeverity, string> = {
  info: "text-[var(--success)] border-[var(--success)]/20 bg-[var(--success)]/10",
  warning: "text-[var(--warning)] border-[var(--warning)]/20 bg-[var(--warning)]/10",
  critical: "text-[var(--error)] border-[var(--error)]/20 bg-[var(--error)]/10",
}

const statusRank: Record<DatabaseHealthStatus, number> = {
  online: 0,
  maintenance: 0,
  degraded: 1,
  offline: 2,
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

function formatDuration(totalSeconds: number): string {
  const safe = Math.max(0, Math.floor(totalSeconds))
  const days = Math.floor(safe / 86400)
  const hours = Math.floor((safe % 86400) / 3600)
  const minutes = Math.floor((safe % 3600) / 60)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
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
          {progress !== undefined && <Progress value={progress} className="h-2" />}
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
  const query = useDatabaseHealth()
  const { data, isLoading, isError, isFetching, refetch } = query
  const loading = isLoading && !data
  const stale = isError && !!data

  const formatNumber = useCallback(
    (value: number, digits: number = 0) =>
      new Intl.NumberFormat(locale, {
        maximumFractionDigits: digits,
        minimumFractionDigits: digits,
      }).format(value),
    [locale]
  )

  const formatDateTime = useCallback(
    (value: string | null | undefined) => {
      if (!value) return "--"
      const date = new Date(value)
      if (Number.isNaN(date.getTime())) return value
      return new Intl.DateTimeFormat(locale, {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
      }).format(date)
    },
    [locale]
  )

  const reasonLabelMap: Record<DatabaseUnavailableReason, string> = {
    permission_denied: t("unavailable.reason.permissionDenied"),
    timeout: t("unavailable.reason.timeout"),
    unsupported: t("unavailable.reason.unsupported"),
    query_failed: t("unavailable.reason.queryFailed"),
    unknown: t("unavailable.reason.unknown"),
  }

  const signalLabelMap = useMemo(
    () => ({
      probeLatencyMs: t("metrics.latencyP95"),
      connectionsMax: t("metrics.connections"),
      lockWaitCount: t("metrics.lockWaitCount"),
      deadlocks1h: t("metrics.deadlocks"),
      longTransactionCount: t("metrics.longTransactions"),
      oldestPendingTaskAgeSec: t("metrics.oldestPendingTaskAge"),
      qps: t("metrics.qps"),
      walGeneratedMb24h: t("metrics.walGenerated"),
      cacheHitRate: t("metrics.cacheHitRate"),
    }),
    [t]
  )

  if (!loading && !data) {
    return (
      <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
        <PageHeader code="DBH-01" title={tPage("title")} description={tPage("description")} />
        <div className="px-4 lg:px-6">
          <Card>
            <CardContent className="py-10 flex flex-col items-center text-center gap-3">
              <IconAlertTriangle className="h-10 w-10 text-destructive" />
              <h3 className="text-base font-semibold">{t("errors.loadFailedTitle")}</h3>
              <p className="text-sm text-muted-foreground">{t("errors.loadFailedDescription")}</p>
              <Button onClick={() => refetch()} disabled={isFetching}>
                {t("errors.retry")}
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    )
  }

  const coreSignals = data?.coreSignals
  const optionalSignals = data?.optionalSignals
  const unavailableSignals = data?.unavailableSignals ?? []
  const fallbackStatus: DatabaseHealthStatus = data?.status ?? "offline"
  const connectionPercent = coreSignals?.connectionUsagePercent ?? 0
  const statusProbe = coreSignals ? statusFromThreshold(coreSignals.probeLatencyMs, 100, 500) : fallbackStatus
  const statusConnections = coreSignals ? statusFromThreshold(connectionPercent, 70, 85) : fallbackStatus
  const statusLockWait = coreSignals ? statusFromThreshold(coreSignals.lockWaitCount, 2, 5) : fallbackStatus
  const statusDeadlocks = coreSignals ? statusFromThreshold(coreSignals.deadlocks1h, 0.5, 1) : fallbackStatus
  const statusLongTransactions =
    coreSignals ? statusFromThreshold(coreSignals.longTransactionCount, 2, 5) : fallbackStatus
  const statusOldestPendingTaskAge =
    coreSignals ? statusFromThreshold(coreSignals.oldestPendingTaskAgeSec, 120, 600) : fallbackStatus
  const statusCache =
    optionalSignals?.cacheHitRate != null
      ? statusFromThreshold(optionalSignals.cacheHitRate, 99, 97, false)
      : "degraded"

  const corePriorityItems = useMemo(
    () => [
      {
        key: "connections",
        label: t("metrics.connections"),
        status: statusConnections,
        value: `${formatNumber(connectionPercent, 1)}%`,
        impact: t("impacts.connections"),
      },
      {
        key: "lockWaitCount",
        label: t("metrics.lockWaitCount"),
        status: statusLockWait,
        value: coreSignals ? formatNumber(coreSignals.lockWaitCount) : "--",
        impact: t("impacts.lockWaitCount"),
      },
      {
        key: "deadlocks1h",
        label: t("metrics.deadlocks"),
        status: statusDeadlocks,
        value: coreSignals ? formatNumber(coreSignals.deadlocks1h, 2) : "--",
        impact: t("impacts.deadlocks1h"),
      },
      {
        key: "oldestPendingTaskAgeSec",
        label: t("metrics.oldestPendingTaskAge"),
        status: statusOldestPendingTaskAge,
        value: coreSignals ? formatDuration(coreSignals.oldestPendingTaskAgeSec) : "--",
        impact: t("impacts.oldestPendingTaskAgeSec"),
      },
    ],
    [
      t,
      statusConnections,
      statusLockWait,
      statusDeadlocks,
      statusOldestPendingTaskAge,
      formatNumber,
      connectionPercent,
      coreSignals,
    ]
  )

  const activeCoreRisks = useMemo(
    () =>
      corePriorityItems
        .filter((item) => item.status !== "online")
        .sort((a, b) => statusRank[b.status] - statusRank[a.status]),
    [corePriorityItems]
  )

  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader code="DBH-01" title={tPage("title")} description={tPage("description")} />

      <div className="px-4 lg:px-6 space-y-6">
        {stale && (
          <Alert variant="destructive">
            <IconAlertTriangle className="h-4 w-4" />
            <AlertTitle>{t("stale.title")}</AlertTitle>
            <AlertDescription>{t("stale.description", { time: formatDateTime(data?.observedAt) })}</AlertDescription>
          </Alert>
        )}

        <Card className={cn(activeCoreRisks.length > 0 && "border-[var(--warning)]/30")}>
          <CardHeader>
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="space-y-1">
                <CardTitle>{t("riskSummary.title")}</CardTitle>
                <CardDescription>{t("riskSummary.description")}</CardDescription>
              </div>
              {loading ? (
                <Skeleton className="h-6 w-24" />
              ) : (
                <Badge variant="outline" className={cn(activeCoreRisks.length > 0 && "text-[var(--warning)]")}>
                  {t("riskSummary.riskCount", { count: activeCoreRisks.length })}
                </Badge>
              )}
            </div>
          </CardHeader>
          <CardContent className="space-y-3">
            {loading ? (
              <div className="grid gap-3 md:grid-cols-2">
                <Skeleton className="h-20 w-full" />
                <Skeleton className="h-20 w-full" />
              </div>
            ) : (
              <>
                <div className="flex flex-wrap items-center gap-3">
                  <Status status={data?.status ?? "offline"} className="px-2 py-1">
                    <StatusIndicator />
                    <StatusLabel>{t(`status.${data?.status ?? "offline"}`)}</StatusLabel>
                  </Status>
                  <span className="text-xs text-muted-foreground">
                    {t("riskSummary.lastCheck", { time: formatDateTime(data?.observedAt) })}
                  </span>
                </div>
                {activeCoreRisks.length > 0 ? (
                  <div className="grid gap-3 md:grid-cols-2">
                    {activeCoreRisks.map((item) => (
                      <div key={item.key} className="rounded-md border p-3">
                        <div className="flex items-center justify-between gap-2">
                          <span className="text-sm font-medium">{item.label}</span>
                          <Status status={item.status} className="px-1.5 py-0.5 text-[10px]">
                            <StatusIndicator />
                            <StatusLabel className="text-[10px]">{t(`status.${item.status}`)}</StatusLabel>
                          </Status>
                        </div>
                        <p className="mt-1 text-sm font-semibold tabular-nums">{item.value}</p>
                        <p className="mt-1 text-xs text-muted-foreground">{item.impact}</p>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-[var(--success)]">{t("riskSummary.noCoreRisk")}</p>
                )}
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-start justify-between gap-4">
            <div className="space-y-1.5">
              <CardTitle>{t("overview.title")}</CardTitle>
              <CardDescription>{t("overview.description")}</CardDescription>
            </div>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
              <SummaryItem
                label={t("overview.role")}
                value={<Badge variant="outline">{t(`overview.roleValue.${data?.role ?? "primary"}`)}</Badge>}
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
              <SummaryItem
                label={t("overview.uptime")}
                value={typeof data?.uptimeSeconds === "number" ? formatDuration(data.uptimeSeconds) : "--"}
                loading={loading}
              />
              <SummaryItem
                label={t("overview.lastCheck")}
                value={data?.observedAt ? formatDateTime(data.observedAt) : "--"}
                loading={loading}
              />
            </div>
          </CardContent>
        </Card>

        <div className="space-y-3">
          <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">{t("sections.core")}</h2>
          <div className="grid grid-cols-1 gap-4 @xl/main:grid-cols-2 @4xl/main:grid-cols-4">
            <MetricCard
              title={t("metrics.connections")}
              value={
                coreSignals
                  ? `${formatNumber(coreSignals.connectionsUsed)} / ${formatNumber(coreSignals.connectionsMax)}`
                  : "--"
              }
              status={statusConnections}
              icon={<IconUsers className="size-4" />}
              loading={loading}
              progress={connectionPercent}
              hint={t("hints.connections", { value: formatNumber(connectionPercent, 1) })}
            />
            <MetricCard
              title={t("metrics.lockWaitCount")}
              value={
                coreSignals ? formatNumber(coreSignals.lockWaitCount) : t("metrics.notAvailable")
              }
              status={statusLockWait}
              icon={<IconDatabase className="size-4" />}
              loading={loading}
              hint={t("hints.lockWaitCount")}
            />
            <MetricCard
              title={t("metrics.deadlocks")}
              value={coreSignals ? formatNumber(coreSignals.deadlocks1h, 2) : t("metrics.notAvailable")}
              status={statusDeadlocks}
              icon={<IconAlertTriangle className="size-4" />}
              loading={loading}
              hint={t("hints.deadlocks")}
            />
            <MetricCard
              title={t("metrics.oldestPendingTaskAge")}
              value={
                coreSignals ? formatDuration(coreSignals.oldestPendingTaskAgeSec) : t("metrics.notAvailable")
              }
              status={statusOldestPendingTaskAge}
              icon={<IconClock className="size-4" />}
              loading={loading}
              hint={t("hints.oldestPendingTaskAge")}
            />
          </div>
        </div>

        <div className="space-y-3">
          <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("sections.coreDetailed")}
          </h2>
          <p className="text-xs text-muted-foreground">{t("sections.coreDetailedDesc")}</p>
          <div className="grid grid-cols-1 gap-4 @xl/main:grid-cols-2">
            <MetricCard
              title={t("metrics.latencyP95")}
              value={coreSignals ? `${formatNumber(coreSignals.probeLatencyMs)} ms` : "--"}
              status={statusProbe}
              icon={<IconTrendingUp className="size-4" />}
              loading={loading}
              hint={t("hints.latencyP95")}
            />
            <MetricCard
              title={t("metrics.longTransactions")}
              value={coreSignals ? formatNumber(coreSignals.longTransactionCount) : t("metrics.notAvailable")}
              status={statusLongTransactions}
              icon={<IconAlertTriangle className="size-4" />}
              loading={loading}
              hint={t("hints.longTransactions")}
            />
          </div>
        </div>

        <div className="grid gap-4 @xl/main:grid-cols-2">
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
                    <div
                      key={alert.id}
                      className="flex items-start justify-between gap-4 border-b pb-3 last:border-b-0 last:pb-0"
                    >
                      <div>
                        <div className="flex items-center gap-2">
                          <Badge variant="outline" className={severityStyles[alert.severity]}>
                            {t(`alerts.severity.${alert.severity}`)}
                          </Badge>
                          <span className="text-sm font-medium">{alert.title}</span>
                        </div>
                        <p className="text-xs text-muted-foreground mt-1">{alert.description}</p>
                      </div>
                      <span className="text-xs text-muted-foreground">{formatDateTime(alert.occurredAt)}</span>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-sm text-muted-foreground">{t("alerts.empty")}</div>
              )}
            </CardContent>
          </Card>

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
                  <div className="flex items-center justify-between gap-4">
                    <div className="text-sm text-muted-foreground">{t("metrics.qps")}</div>
                    <span className="text-sm font-medium tabular-nums">
                      {optionalSignals?.qps != null ? formatNumber(optionalSignals.qps) : t("metrics.notAvailable")}
                    </span>
                  </div>
                  <div className="flex items-center justify-between gap-4">
                    <div className="text-sm text-muted-foreground">{t("metrics.walGenerated")}</div>
                    <span className="text-sm font-medium tabular-nums">
                      {optionalSignals?.walGeneratedMb24h != null
                        ? `${formatNumber(optionalSignals.walGeneratedMb24h)} MB`
                        : t("metrics.notAvailable")}
                    </span>
                  </div>
                  <div className="flex items-center justify-between gap-4">
                    <div className="text-sm text-muted-foreground">{t("metrics.cacheHitRate")}</div>
                    <div className="flex items-center gap-3">
                      <span className="text-sm font-medium tabular-nums">
                        {optionalSignals?.cacheHitRate != null
                          ? `${formatNumber(optionalSignals.cacheHitRate, 1)}%`
                          : t("metrics.notAvailable")}
                      </span>
                      <Status status={statusCache} className="px-1.5 py-0.5 text-[10px]">
                        <StatusIndicator />
                        <StatusLabel className="text-[10px]">{t(`status.${statusCache}`)}</StatusLabel>
                      </Status>
                    </div>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {!!unavailableSignals.length && (
          <Card>
            <CardHeader>
              <CardTitle>{t("unavailable.title")}</CardTitle>
              <CardDescription>{t("unavailable.description")}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2">
              {unavailableSignals.map((signal) => (
                <div
                  key={`${signal.scope}:${signal.name}:${signal.reasonCode}`}
                  className="flex items-center justify-between gap-3 text-sm border-b pb-2 last:border-b-0 last:pb-0"
                >
                  <div className="flex items-center gap-2">
                    <Badge variant="outline">{t(`unavailable.scope.${signal.scope}`)}</Badge>
                    <span>{signalLabelMap[signal.name as keyof typeof signalLabelMap] ?? signal.name}</span>
                  </div>
                  <span className="text-muted-foreground">{reasonLabelMap[signal.reasonCode] ?? signal.reasonCode}</span>
                </div>
              ))}
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}
