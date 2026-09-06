"use client"

import { useCallback, useEffect, useMemo, type ReactNode } from "react"
import { useLocale, useTranslations } from "next-intl"
import { IconAlertTriangle } from "@/components/icons"

import { useDatabaseHealth } from "@/hooks/use-database-health"
import type {
  DatabaseHealthAlert,
  DatabaseAlertSeverity,
  DatabaseHealthFinding,
  DatabaseHealthStatus,
  DatabaseUnavailableReason,
} from "@/types/database-health.types"
import { PageHeader } from "@/components/common/page-header"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import {
  TABLE_DENSE_CELL_RHYTHM_CLASS,
  TABLE_DENSE_ROW_CLASS,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  getStatusToneBadgeClass,
  getStatusToneTextClass,
  type StatusTone,
} from "@/lib/status-config"
import { cn, formatBytes } from "@/lib/utils"
import { normalizeError } from "@/lib/errors/normalize-error"
import { textRole } from "@/lib/typography"
import {
  DatabaseHealthLoadingState,
  DatabaseHealthSectionPanel as SectionPanel,
} from "@/components/settings/database-health/database-health-loading-state"
import {
  DATABASE_HEALTH_CONTENT_SHELL_CLASS,
  DATABASE_HEALTH_CONTEXT_STAT_CLASS,
  DATABASE_HEALTH_CORE_METRIC_CLASS,
  DATABASE_HEALTH_CORE_METRIC_DETAIL_ROW_CLASS,
  DATABASE_HEALTH_CORE_METRIC_GRID_CLASS,
  DATABASE_HEALTH_FINDING_DETAIL_GRID_CLASS,
  DATABASE_HEALTH_FINDING_HEADER_CLASS,
  DATABASE_HEALTH_FINDING_ROW_CLASS,
  DATABASE_HEALTH_FINDING_TITLE_GROUP_CLASS,
  DATABASE_HEALTH_ALERT_TABLE_COLUMN_COUNT,
  DATABASE_HEALTH_OPTIONAL_SIGNAL_TABLE_COLUMN_COUNT,
  DATABASE_HEALTH_PAGE_SHELL_CLASS,
  DATABASE_HEALTH_SECONDARY_GRID_CLASS,
  DATABASE_HEALTH_SECTION_BODY_FLUSH_CLASS,
  DATABASE_HEALTH_SECTION_HEADER_CLASS,
  DATABASE_HEALTH_SECTION_HEADER_WITH_DESCRIPTION_CLASS,
  DATABASE_HEALTH_SNAPSHOT_AUTO_CHECK_CLASS,
  DATABASE_HEALTH_SNAPSHOT_GRID_CLASS,
  DATABASE_HEALTH_SNAPSHOT_HEADER_CLASS,
  DATABASE_HEALTH_SNAPSHOT_STATUS_GROUP_CLASS,
  DATABASE_HEALTH_TABLE_CONTENT_CLASS,
} from "@/components/settings/database-health/database-health-layout"

export interface DatabaseHealthViewProps {
  pageTitle: string
  pageDescription: string
  onReady?: () => void
  deferInitialSkeleton?: boolean
}

const AUTO_CHECK_INTERVAL_SECONDS = 60
const PENDING_TASK_BACKLOG_TARGET_SECONDS = 120
const PENDING_TASK_BACKLOG_CRITICAL_SECONDS = 600
type CoreMetricKey =
  | "oldestPendingTaskAgeSec"
  | "connections"
  | "lockWaitCount"
  | "deadlocks1h"
  | "probeLatencyMs"
  | "longTransactionCount"

interface ContextItem {
  label: string
  value: ReactNode
}

interface CoreMetricItem {
  key: CoreMetricKey
  label: string
  value: string
  target: string
  status: DatabaseHealthStatus
  progress?: number
  progressLabel?: string
}

interface OptionalSignalRow {
  id: string
  label: string
  value: string
  target: string
  status: DatabaseHealthStatus
}

function getDatabaseAlertTone(severity: DatabaseAlertSeverity): StatusTone {
  if (severity === "info") return "success"
  if (severity === "warning") return "warning"
  return "error"
}

function getDatabaseHealthTone(status: DatabaseHealthStatus): StatusTone {
  if (status === "online") return "success"
  if (status === "degraded") return "warning"
  if (status === "offline") return "error"
  return "muted"
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

function formatDurationUnit(locale: string, value: number, unit: Intl.NumberFormatOptions["unit"]): string {
  return new Intl.NumberFormat(locale, {
    style: "unit",
    unit,
    unitDisplay: "narrow",
    maximumFractionDigits: 0,
  }).format(value)
}

function formatDurationSeconds(totalSeconds: number, locale: string): string {
  const safe = Math.max(0, Math.floor(totalSeconds))
  const days = Math.floor(safe / 86400)
  const hours = Math.floor((safe % 86400) / 3600)
  const minutes = Math.floor((safe % 3600) / 60)
  const seconds = safe % 60
  if (days > 0) {
    return [formatDurationUnit(locale, days, "day"), hours > 0 ? formatDurationUnit(locale, hours, "hour") : null]
      .filter(Boolean)
      .join(" ")
  }
  if (hours > 0) {
    return [formatDurationUnit(locale, hours, "hour"), minutes > 0 ? formatDurationUnit(locale, minutes, "minute") : null]
      .filter(Boolean)
      .join(" ")
  }
  if (minutes > 0) {
    return [formatDurationUnit(locale, minutes, "minute"), seconds > 0 ? formatDurationUnit(locale, seconds, "second") : null]
      .filter(Boolean)
      .join(" ")
  }
  return formatDurationUnit(locale, seconds, "second")
}

function normalizePostgresVersion(version: string | undefined): string {
  if (!version) return "--"
  return version.toLowerCase().startsWith("postgresql") ? version : `PostgreSQL ${version}`
}

function ContextStat({ label, value }: ContextItem) {
  return (
    <div className={DATABASE_HEALTH_CONTEXT_STAT_CLASS}>
      <div className={cn("whitespace-nowrap tabular-nums", textRole.metadataValueStrong)}>{value}</div>
      <div className={cn("mt-1 truncate", textRole.caption)}>{label}</div>
    </div>
  )
}

function MetricStripItem({ item }: { item: CoreMetricItem }) {
  const tone = getDatabaseHealthTone(item.status)
  const unhealthy = item.status !== "online"

  return (
    <div
      className={cn(
        DATABASE_HEALTH_CORE_METRIC_CLASS,
        unhealthy && "bg-secondary/30"
      )}
    >
      <div className={cn(textRole.tableCellSecondary, unhealthy && getStatusToneTextClass(tone))}>
        {item.label}
      </div>
      <div
        className={cn(
          "mt-2 tabular-nums",
          textRole.metricValueDisplay,
          unhealthy && getStatusToneTextClass(tone)
        )}
      >
        {item.value}
      </div>
      {item.progress !== undefined ? (
        <div className="mt-2 max-w-44">
          <Progress value={item.progress} className="h-2" indicatorClassName="bg-success" />
        </div>
      ) : null}
      <div className={cn(DATABASE_HEALTH_CORE_METRIC_DETAIL_ROW_CLASS, textRole.caption)}>
        {item.progressLabel ? <span>{item.progressLabel}</span> : null}
        <span>{item.target}</span>
      </div>
    </div>
  )
}

function SeverityBadge({ severity }: { severity: DatabaseAlertSeverity }) {
  const t = useTranslations("databaseHealth")
  return (
    <Badge variant="outline" className={getStatusToneBadgeClass(getDatabaseAlertTone(severity))}>
      {t(`alerts.severity.${severity}`)}
    </Badge>
  )
}

function FindingCard({ finding }: { finding: DatabaseHealthFinding }) {
  const t = useTranslations("databaseHealth")
  const tone = getDatabaseAlertTone(finding.severity)
  const translatedFinding = translateDatabaseHealthFinding(finding, t)

  return (
    <div className={DATABASE_HEALTH_FINDING_ROW_CLASS}>
      <div className={DATABASE_HEALTH_FINDING_HEADER_CLASS}>
        <div className="min-w-0">
          <div className={DATABASE_HEALTH_FINDING_TITLE_GROUP_CLASS}>
            <Badge variant="outline" className={getStatusToneBadgeClass(tone)}>
              {t(`alerts.severity.${translatedFinding.severity}`)}
            </Badge>
            <span className={cn("break-words", textRole.tableCellPrimary)}>{translatedFinding.title}</span>
          </div>
          <p className={cn("mt-2", textRole.bodySubtle)}>{translatedFinding.description}</p>
        </div>
        <Badge variant="outline" className="shrink-0">
          {translatedFinding.signal}
        </Badge>
      </div>
      <div className={DATABASE_HEALTH_FINDING_DETAIL_GRID_CLASS}>
        <div className="min-w-0">
          <div className={textRole.caption}>{t("findings.evidence")}</div>
          <div className="mt-1 flex flex-wrap gap-2">
            {translatedFinding.evidence.map((item) => (
              <Badge key={item} variant="secondary" className="max-w-full">
                <span className="truncate">{item}</span>
              </Badge>
            ))}
          </div>
        </div>
        <div className="min-w-0">
          <div className={textRole.caption}>{t("findings.recommendation")}</div>
          <p className={cn("mt-1", textRole.body)}>{translatedFinding.recommendation}</p>
        </div>
      </div>
    </div>
  )
}

function translateDatabaseHealthFinding(
  finding: DatabaseHealthFinding,
  t: ReturnType<typeof useTranslations<"databaseHealth">>
): DatabaseHealthFinding {
  if (finding.signal === "connectionUsagePercent") {
    return {
      ...finding,
      title: t("backendFindings.connectionUsagePercent.title"),
      description: t("backendFindings.connectionUsagePercent.description"),
      recommendation: t("backendFindings.connectionUsagePercent.recommendation"),
    }
  }
  if (finding.signal !== "oldestPendingTaskAgeSec") return finding
  return {
    ...finding,
    title: t("backendFindings.oldestPendingTaskAgeSec.title"),
    description: t("backendFindings.oldestPendingTaskAgeSec.description"),
    recommendation: t("backendFindings.oldestPendingTaskAgeSec.recommendation"),
  }
}

function translateDatabaseHealthAlert(
  alert: DatabaseHealthAlert,
  t: ReturnType<typeof useTranslations<"databaseHealth">>
): DatabaseHealthAlert {
  if (alert.title === "Established connection count is high") {
    return {
      ...alert,
      title: t("backendAlerts.establishedConnectionCountHigh.title"),
      description: t("backendAlerts.establishedConnectionCountHigh.description"),
    }
  }
  if (alert.title === "Task backlog age critical") {
    return {
      ...alert,
      title: t("backendAlerts.taskBacklogAgeCritical.title"),
      description: t("backendAlerts.taskBacklogAgeCritical.description"),
    }
  }
  if (alert.title === "Task backlog age high") {
    return {
      ...alert,
      title: t("backendAlerts.taskBacklogAgeHigh.title"),
      description: t("backendAlerts.taskBacklogAgeHigh.description"),
    }
  }
  return alert
}

function SignalStatusText({ status }: { status: DatabaseHealthStatus }) {
  const t = useTranslations("databaseHealth")
  return (
    <span className={cn(textRole.tableCellPrimary, getStatusToneTextClass(getDatabaseHealthTone(status)))}>
      {status === "online" ? t("signals.normal") : t(`status.${status}`)}
    </span>
  )
}

export function DatabaseHealthView({
  pageTitle,
  pageDescription,
  onReady,
  deferInitialSkeleton = false,
}: DatabaseHealthViewProps) {
  const t = useTranslations("databaseHealth")
  const locale = useLocale()
  const query = useDatabaseHealth()
  const { data, isLoading, isError, refetch } = query
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
        second: "2-digit",
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

  const coreSignals = data?.coreSignals
  const optionalSignals = data?.optionalSignals
  const unavailableSignals = data?.unavailableSignals ?? []
  const findings = data?.findings ?? []
  const alerts = useMemo(() => data?.alerts ?? [], [data?.alerts])
  const fallbackStatus: DatabaseHealthStatus = data?.status ?? "offline"
  const connectionPercent = coreSignals?.connectionUsagePercent ?? 0
  const statusProbe = coreSignals ? statusFromThreshold(coreSignals.probeLatencyMs, 100, 500) : fallbackStatus
  const statusConnections = coreSignals ? statusFromThreshold(connectionPercent, 70, 85) : fallbackStatus
  const statusLockWait = coreSignals ? statusFromThreshold(coreSignals.lockWaitCount, 2, 5) : fallbackStatus
  const statusDeadlocks = coreSignals ? statusFromThreshold(coreSignals.deadlocks1h, 0.5, 1) : fallbackStatus
  const statusLongTransactions =
    coreSignals ? statusFromThreshold(coreSignals.longTransactionCount, 2, 5) : fallbackStatus
  const statusOldestPendingTaskAge =
    coreSignals
      ? statusFromThreshold(
          coreSignals.oldestPendingTaskAgeSec,
          PENDING_TASK_BACKLOG_TARGET_SECONDS,
          PENDING_TASK_BACKLOG_CRITICAL_SECONDS
        )
      : fallbackStatus
  const statusCache =
    optionalSignals?.cacheHitRate != null
      ? statusFromThreshold(optionalSignals.cacheHitRate, 99, 97, false)
      : "maintenance"

  const coreMetrics = useMemo<CoreMetricItem[]>(
    () => [
      {
        key: "oldestPendingTaskAgeSec",
        label: t("metrics.oldestPendingTaskAge"),
        status: statusOldestPendingTaskAge,
        value: coreSignals ? formatDurationSeconds(coreSignals.oldestPendingTaskAgeSec, locale) : "--",
        target: t("hints.oldestPendingTaskAge"),
      },
      {
        key: "connections",
        label: t("metrics.connections"),
        status: statusConnections,
        value: coreSignals
          ? `${formatNumber(coreSignals.connectionsUsed)} / ${formatNumber(coreSignals.connectionsMax)}`
          : "--",
        target: t("targets.connections"),
        progress: connectionPercent,
        progressLabel: t("hints.connections", { value: formatNumber(connectionPercent, 1) }),
      },
      {
        key: "lockWaitCount",
        label: t("metrics.lockWaitCount"),
        status: statusLockWait,
        value: coreSignals ? formatNumber(coreSignals.lockWaitCount) : "--",
        target: t("hints.lockWaitCount"),
      },
      {
        key: "deadlocks1h",
        label: t("metrics.deadlocks"),
        status: statusDeadlocks,
        value: coreSignals ? formatNumber(coreSignals.deadlocks1h) : "--",
        target: t("targets.deadlocks"),
      },
      {
        key: "probeLatencyMs",
        label: t("metrics.probeLatency"),
        status: statusProbe,
        value: coreSignals ? `${formatNumber(coreSignals.probeLatencyMs)} ms` : "--",
        target: t("hints.probeLatency"),
      },
      {
        key: "longTransactionCount",
        label: t("metrics.longTransactions"),
        status: statusLongTransactions,
        value: coreSignals ? formatNumber(coreSignals.longTransactionCount) : "--",
        target: t("hints.longTransactions"),
      },
    ],
    [
      t,
      statusOldestPendingTaskAge,
      statusConnections,
      statusLockWait,
      statusDeadlocks,
      statusProbe,
      statusLongTransactions,
      coreSignals,
      formatNumber,
      connectionPercent,
      locale,
    ]
  )

  const contextItems = useMemo<ContextItem[]>(
    () => [
      {
        label: t("overview.version"),
        value: normalizePostgresVersion(data?.version),
      },
      {
        label: t("overview.role"),
        value: t(`overview.roleValue.${data?.role ?? "primary"}`),
      },
      {
        label: t("overview.readOnly"),
        value: data?.readOnly ? t("overview.readOnlyYes") : t("overview.readOnlyNo"),
      },
      {
        label: t("overview.uptime"),
        value: typeof data?.uptimeSeconds === "number" ? formatDurationSeconds(data.uptimeSeconds, locale) : "--",
      },
      {
        label: t("overview.databaseSize"),
        value: data?.databaseSizeBytes != null ? formatBytes(data.databaseSizeBytes, 0) : "--",
      },
      {
        label: t("overview.lastCheck"),
        value: data?.observedAt ? formatDateTime(data.observedAt) : "--",
      },
    ],
    [data, formatDateTime, locale, t]
  )

  const translatedAlerts = useMemo(
    () => alerts.map((alert) => translateDatabaseHealthAlert(alert, t)),
    [alerts, t]
  )

  const optionalSignalRows = useMemo<OptionalSignalRow[]>(
    () => [
      {
        id: "qps",
        label: t("metrics.qps"),
        value: optionalSignals?.qps != null ? formatNumber(optionalSignals.qps) : "--",
        target: "--",
        status: optionalSignals?.qps != null ? "online" : "maintenance",
      },
      {
        id: "walGeneratedMb24h",
        label: t("metrics.walGenerated"),
        value: optionalSignals?.walGeneratedMb24h != null ? `${formatNumber(optionalSignals.walGeneratedMb24h)} MB` : "--",
        target: "--",
        status: optionalSignals?.walGeneratedMb24h != null ? "online" : "maintenance",
      },
      {
        id: "cacheHitRate",
        label: t("metrics.cacheHitRate"),
        value: optionalSignals?.cacheHitRate != null ? `${formatNumber(optionalSignals.cacheHitRate, 1)}%` : "--",
        target: t("hints.cacheHitRate"),
        status: statusCache,
      },
    ],
    [formatNumber, optionalSignals, statusCache, t]
  )

  useEffect(() => {
    if (loading) return
    onReady?.()
  }, [loading, onReady])

  if (loading && !deferInitialSkeleton) {
    return (
      <DatabaseHealthLoadingState
        owner="database-health-page"
        pageTitle={pageTitle}
        pageDescription={pageDescription}
      />
    )
  }
  if (loading) return null

  if (!loading && !data) {
    return (
      <div className={DATABASE_HEALTH_PAGE_SHELL_CLASS}>
        <div {...getLoadingStructureSlotAttributes("database-health-header")}>
          <PageHeader code="DBH-01" title={pageTitle} description={pageDescription} />
        </div>
        <div className={DATABASE_HEALTH_CONTENT_SHELL_CLASS}>
          <AppErrorState
            error={normalizeError(query.error, { notFoundKind: "unexpected-error" })}
            title={t("errors.loadFailedTitle")}
            description={t("errors.loadFailedDescription")}
            onRetry={refetch}
            variant="section"
          />
        </div>
      </div>
    )
  }

  return (
    <div className={DATABASE_HEALTH_PAGE_SHELL_CLASS}>
      <div {...getLoadingStructureSlotAttributes("database-health-header")}>
        <PageHeader code="DBH-01" title={pageTitle} description={pageDescription} />
      </div>

      <div className={DATABASE_HEALTH_CONTENT_SHELL_CLASS}>
        {stale && (
          <Alert variant="destructive">
            <IconAlertTriangle className="h-4 w-4" />
            <AlertTitle>{t("stale.title")}</AlertTitle>
            <AlertDescription>{t("stale.description", { time: formatDateTime(data?.observedAt) })}</AlertDescription>
          </Alert>
        )}

        <SectionPanel {...getLoadingStructureSlotAttributes("database-health-snapshot")}>
          <CardHeader className={DATABASE_HEALTH_SNAPSHOT_HEADER_CLASS}>
            <div className="min-w-0">
              <CardTitle>{t("sections.snapshot")}</CardTitle>
              <CardDescription className="mt-1">{t("sections.snapshotDesc")}</CardDescription>
            </div>
            <div className={DATABASE_HEALTH_SNAPSHOT_STATUS_GROUP_CLASS}>
              <span className={DATABASE_HEALTH_SNAPSHOT_AUTO_CHECK_CLASS}>
                <span className="size-2 rounded-full bg-success" aria-hidden="true" />
                <span className={textRole.metadataLabel}>
                  {t("instance.autoCheckValue", { seconds: AUTO_CHECK_INTERVAL_SECONDS })}
                  <span className="ml-1">{t("instance.autoCheck")}</span>
                </span>
              </span>
            </div>
          </CardHeader>
          <CardContent className={DATABASE_HEALTH_SNAPSHOT_GRID_CLASS}>
            {contextItems.map((item) => (
              <ContextStat key={item.label} {...item} />
            ))}
          </CardContent>
        </SectionPanel>

        <SectionPanel {...getLoadingStructureSlotAttributes("database-health-metrics")}>
          <CardHeader className={DATABASE_HEALTH_SECTION_HEADER_CLASS}>
            <CardTitle>{t("sections.core")}</CardTitle>
          </CardHeader>
          <CardContent className={DATABASE_HEALTH_SECTION_BODY_FLUSH_CLASS}>
            <div className={DATABASE_HEALTH_CORE_METRIC_GRID_CLASS}>
              {coreMetrics.map((item) => (
                <MetricStripItem key={item.key} item={item} />
              ))}
            </div>
          </CardContent>
        </SectionPanel>

        <SectionPanel
          {...getLoadingStructureSlotAttributes("database-health-findings")}
          id="database-health-findings"
        >
          <CardHeader className={DATABASE_HEALTH_SECTION_HEADER_CLASS}>
            <CardTitle>{t("findings.title")}</CardTitle>
          </CardHeader>
          <CardContent className={DATABASE_HEALTH_SECTION_BODY_FLUSH_CLASS}>
            {findings.length ? (
              findings.map((finding) => (
                <FindingCard key={`${finding.signal}:${finding.title}`} finding={finding} />
              ))
            ) : (
              <div className={cn("p-4", textRole.bodySubtle)}>{t("findings.empty")}</div>
            )}
          </CardContent>
        </SectionPanel>

        <div className={DATABASE_HEALTH_SECONDARY_GRID_CLASS}>
          <SectionPanel id="database-health-optional-signals">
            <CardHeader className={DATABASE_HEALTH_SECTION_HEADER_CLASS}>
              <CardTitle>{t("sections.optional")}</CardTitle>
            </CardHeader>
            <CardContent className={DATABASE_HEALTH_TABLE_CONTENT_CLASS}>
              <Table aria-colcount={DATABASE_HEALTH_OPTIONAL_SIGNAL_TABLE_COLUMN_COUNT}>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("signals.columns.signal")}</TableHead>
                    <TableHead>{t("signals.columns.current")}</TableHead>
                    <TableHead>{t("signals.columns.target")}</TableHead>
                    <TableHead>{t("signals.columns.status")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {optionalSignalRows.map((row) => (
                    <TableRow key={row.id} className={TABLE_DENSE_ROW_CLASS}>
                      <TableCell className={cn(TABLE_DENSE_CELL_RHYTHM_CLASS, textRole.tableCellSecondary)}>
                        {row.label}
                      </TableCell>
                      <TableCell className={cn(TABLE_DENSE_CELL_RHYTHM_CLASS, "tabular-nums", textRole.tableCellPrimary)}>
                        {row.value}
                      </TableCell>
                      <TableCell className={cn(TABLE_DENSE_CELL_RHYTHM_CLASS, textRole.tableCellSecondary)}>
                        {row.target}
                      </TableCell>
                      <TableCell className={TABLE_DENSE_CELL_RHYTHM_CLASS}>
                        <SignalStatusText status={row.status} />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </SectionPanel>

          <SectionPanel id="database-health-alerts">
            <CardHeader className={DATABASE_HEALTH_SECTION_HEADER_CLASS}>
              <CardTitle>{t("alerts.title")}</CardTitle>
            </CardHeader>
            <CardContent className={DATABASE_HEALTH_TABLE_CONTENT_CLASS}>
              {translatedAlerts.length ? (
                <Table aria-colcount={DATABASE_HEALTH_ALERT_TABLE_COLUMN_COUNT}>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t("alerts.columns.severity")}</TableHead>
                      <TableHead>{t("alerts.columns.name")}</TableHead>
                      <TableHead>{t("alerts.columns.description")}</TableHead>
                      <TableHead>{t("alerts.columns.observedAt")}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {translatedAlerts.map((alert) => (
                      <TableRow key={alert.id} className={TABLE_DENSE_ROW_CLASS}>
                        <TableCell className={TABLE_DENSE_CELL_RHYTHM_CLASS}>
                          <SeverityBadge severity={alert.severity} />
                        </TableCell>
                        <TableCell className={cn(TABLE_DENSE_CELL_RHYTHM_CLASS, textRole.tableCellPrimary)}>
                          {alert.title}
                        </TableCell>
                        <TableCell className={cn(TABLE_DENSE_CELL_RHYTHM_CLASS, textRole.tableCellSecondary)}>
                          {alert.description}
                        </TableCell>
                        <TableCell className={cn(TABLE_DENSE_CELL_RHYTHM_CLASS, "whitespace-nowrap tabular-nums", textRole.tableCellSecondary)}>
                          {formatDateTime(alert.occurredAt)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <div className={cn("p-4", textRole.bodySubtle)}>{t("alerts.empty")}</div>
              )}
            </CardContent>
          </SectionPanel>
        </div>

        {!!unavailableSignals.length && (
          <SectionPanel>
            <CardHeader className={DATABASE_HEALTH_SECTION_HEADER_WITH_DESCRIPTION_CLASS}>
              <CardTitle>{t("unavailable.title")}</CardTitle>
              <CardDescription>{t("unavailable.description")}</CardDescription>
            </CardHeader>
            <CardContent className={DATABASE_HEALTH_TABLE_CONTENT_CLASS}>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("unavailable.columns.scope")}</TableHead>
                    <TableHead>{t("unavailable.columns.signal")}</TableHead>
                    <TableHead>{t("unavailable.columns.reason")}</TableHead>
                    <TableHead>{t("unavailable.columns.message")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {unavailableSignals.map((signal) => (
                    <TableRow key={`${signal.scope}:${signal.name}:${signal.reasonCode}`} className={TABLE_DENSE_ROW_CLASS}>
                      <TableCell className={TABLE_DENSE_CELL_RHYTHM_CLASS}>
                        <Badge variant="outline">{t(`unavailable.scope.${signal.scope}`)}</Badge>
                      </TableCell>
                      <TableCell className={cn(TABLE_DENSE_CELL_RHYTHM_CLASS, textRole.tableCellPrimary)}>
                        {signal.name}
                      </TableCell>
                      <TableCell className={cn(TABLE_DENSE_CELL_RHYTHM_CLASS, textRole.tableCellSecondary)}>
                        {reasonLabelMap[signal.reasonCode] ?? signal.reasonCode}
                      </TableCell>
                      <TableCell className={cn(TABLE_DENSE_CELL_RHYTHM_CLASS, textRole.tableCellSecondary)}>
                        {signal.message ?? "--"}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </SectionPanel>
        )}
      </div>
    </div>
  )
}
