"use client"

import React from "react"
import { useTranslations } from "next-intl"
import { CheckCircle2, XCircle, AlertCircle, Clock, RefreshCw } from "@/components/icons"
import { RefreshSpinner, Spinner } from "@/components/shared/loading/spinner"
import { Button } from "@/components/ui/button"
import {
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
} from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import { useScheduledScanOccurrences } from "@/hooks/use-scheduled-scans"
import type {
  ScheduledScanOccurrence,
  ScheduledScanOccurrenceStatus,
} from "@/types/scheduled-scan.types"

// Matches the server handoff deadline; a DISPATCHING row older than this is
// surfaced as suspected-stuck instead of silently looking in-flight.
const SUSPECTED_STUCK_AFTER_MS = 5 * 60 * 1000
const FAILURE_CAUSE_TONE: Record<ScheduledScanOccurrenceStatus, "success" | "error" | "warning" | "info" | "muted"> = {
  SUCCEEDED: "success",
  FAILED: "error",
  RETRYING: "warning",
  DISPATCHING: "info",
  PENDING: "muted",
}

interface ScheduledScanOccurrencesTabProps {
  scheduledScanId: number
  formatDate: (dateString: string) => string
}

export function ScheduledScanOccurrencesTab({
  scheduledScanId,
  formatDate,
}: ScheduledScanOccurrencesTabProps) {
  const t = useTranslations("scan.scheduled.occurrences")
  const tColumns = useTranslations("columns.scheduledScan")
  const query = useScheduledScanOccurrences(scheduledScanId)
  const data = query.data
  const isInitialLoading = query.isPending
  const isRefreshing = query.isFetching && !isInitialLoading

  if (isInitialLoading) {
    return (
      <div className="flex h-48 items-center justify-center">
        <Spinner className="size-5" />
      </div>
    )
  }

  if (query.isError && data === undefined) {
    return (
      <div className="flex h-48 flex-col items-center justify-center gap-2 text-center">
        <AlertCircle className="size-8 stroke-1 text-muted-foreground/50" />
        <p className={cn(textRole.bodySubtle, "text-error")}>{t("loadFailed")}</p>
        <Button type="button" variant="outline" size="sm" onClick={() => void query.refetch()}>
          {t("loadFailedRetry")}
        </Button>
      </div>
    )
  }

  if (!data) return null

  const { occurrences, statusCounts, totalSize } = data
  const hasRetrying = statusCounts.retrying > 0
  const hasDispatching = statusCounts.dispatching > 0
  const hasPending = statusCounts.pending > 0

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden">
      <div className="flex items-center justify-between border-b pb-2.5">
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className={textRole.compactCaption}>{t("summary.retentionBounded")}</span>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <OccurrenceSummaryValue label={t("summary.total")} value={totalSize ?? occurrences.length} tone="muted" />
            <SummaryDivider />
            <OccurrenceSummaryValue label={t("summary.succeeded")} value={statusCounts.succeeded} tone="success" />
            <SummaryDivider />
            <OccurrenceSummaryValue label={t("summary.failed")} value={statusCounts.failed} tone="error" />
            {hasRetrying && (
              <>
                <SummaryDivider />
                <OccurrenceSummaryValue label={t("summary.retrying")} value={statusCounts.retrying} tone="warning" />
              </>
            )}
            {hasDispatching && (
              <>
                <SummaryDivider />
                <OccurrenceSummaryValue label={t("summary.dispatching")} value={statusCounts.dispatching} tone="info" />
              </>
            )}
            {hasPending && (
              <>
                <SummaryDivider />
                <OccurrenceSummaryValue label={t("summary.pending")} value={statusCounts.pending} tone="muted" />
              </>
            )}
          </div>
        </div>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-7 shrink-0 px-2 text-xs text-muted-foreground hover:text-foreground"
          onClick={() => void query.refetch()}
          disabled={isRefreshing}
        >
          {isRefreshing ? (
            <RefreshSpinner className="mr-1 size-3.5" aria-hidden="true" />
          ) : (
            <RefreshCw className="mr-1 size-3.5" aria-hidden="true" />
          )}
          {t("refresh")}
        </Button>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto rounded-md border bg-card">
        {occurrences.length === 0 ? (
          <div className="flex flex-col items-center justify-center p-8 text-center text-muted-foreground">
            <Clock className="mb-2 size-8 stroke-1 text-muted-foreground/50" />
            <p className={textRole.bodyStrong}>{t("empty.title")}</p>
            <p className={cn(textRole.helperText, "mt-1 max-w-sm")}>{t("empty.description")}</p>
          </div>
        ) : (
          <Table>
            <TableHeader className="sticky top-0 z-10 bg-muted/40 backdrop-blur-sm">
              <TableRow className="hover:bg-transparent">
                <TableHead className={cn(textRole.tableHeader, "w-[170px]")}>{t("columns.scheduledFor")}</TableHead>
                <TableHead className={cn(textRole.tableHeader, "w-[120px]")}>{t("columns.status")}</TableHead>
                <TableHead className={textRole.tableHeader}>{t("columns.result")}</TableHead>
                <TableHead className={cn(textRole.tableHeader, "w-[90px] text-right")}>{t("columns.duration")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {occurrences.map((occurrence) => (
                <OccurrenceRow
                  key={occurrence.id}
                  occurrence={occurrence}
                  formatDate={formatDate}
                  t={t}
                  tColumns={tColumns}
                />
              ))}
            </TableBody>
          </Table>
        )}
      </div>
    </div>
  )
}

function OccurrenceSummaryValue({ label, value, tone }: { label: string; value: number; tone: "success" | "error" | "warning" | "info" | "muted" }) {
  return (
    <span className="inline-flex items-center gap-1.5 font-mono tabular-nums">
      <span className={cn(textRole.compactCaption, getStatusToneTextClass(tone))}>{label}</span>
      <span className={cn(textRole.tableCellPrimary, getStatusToneTextClass(tone))}>{value}</span>
    </span>
  )
}

function SummaryDivider() {
  return <span className="text-border" aria-hidden="true">|</span>
}

function OccurrenceRow({
  occurrence,
  formatDate,
  t,
  tColumns,
}: {
  occurrence: ScheduledScanOccurrence
  formatDate: (dateString: string) => string
  t: ReturnType<typeof useTranslations<"scan.scheduled.occurrences">>
  tColumns: ReturnType<typeof useTranslations<"columns.scheduledScan">>
}) {
  const tone = FAILURE_CAUSE_TONE[occurrence.status]
  const isSuspectedStuck =
    occurrence.status === "DISPATCHING" &&
    occurrence.attemptedAt !== null &&
    Date.now() - Date.parse(occurrence.attemptedAt) > SUSPECTED_STUCK_AFTER_MS

  return (
    <TableRow className="text-xs hover:bg-muted/30">
      <TableCell className={cn(textRole.tableCellSecondary, "py-2.5 font-mono")}>
        {formatDate(occurrence.scheduledFor)}
      </TableCell>

      <TableCell className="py-2.5">
        <span className={cn("inline-flex items-center gap-1 rounded-full border px-2 py-0.5", textRole.badge, getStatusToneBadgeClass(tone))}>
          {occurrence.status === "SUCCEEDED" && <CheckCircle2 className="size-3" />}
          {occurrence.status === "FAILED" && <XCircle className="size-3" />}
          {t(`status.${occurrence.status}`)}
        </span>
      </TableCell>

      <TableCell className="py-2.5">
        {occurrence.status === "SUCCEEDED" && (
          <span className={textRole.tableCellPrimary}>{t("result.succeededHandoff")}</span>
        )}
        {occurrence.status === "FAILED" && (
          <div className="flex flex-col gap-1">
            <div className="flex flex-wrap items-center gap-1.5">
              {occurrence.failureCause && (
                <span className={cn("inline-flex items-center gap-1 rounded border px-1.5 py-0.5", textRole.badge, getStatusToneBadgeClass("error"))}>
                  <AlertCircle className="size-3" />
                  {tColumns(`failureCauses.${occurrence.failureCause}`)}
                </span>
              )}
              <span className={cn(textRole.compactCaption, "font-mono")}>
                {t("result.retryExhausted", { current: occurrence.retryCount, max: 3 })}
              </span>
            </div>
            <FailureMessage message={occurrence.failureMessage} t={t} />
          </div>
        )}
        {occurrence.status === "RETRYING" && (
          <div className="flex flex-col gap-1">
            <div className="flex flex-wrap items-center gap-1.5">
              {occurrence.failureCause && (
                <span className={cn("inline-flex items-center gap-1 rounded border px-1.5 py-0.5", textRole.badge, getStatusToneBadgeClass("warning"))}>
                  {tColumns(`failureCauses.${occurrence.failureCause}`)}
                </span>
              )}
              <span className={cn(textRole.compactCaption, getStatusToneTextClass("warning"))}>
                {t("result.retryProgress", { current: occurrence.retryCount, max: 3 })} · {t("result.awaitRetry")}
              </span>
            </div>
            <FailureMessage message={occurrence.failureMessage} t={t} />
          </div>
        )}
        {occurrence.status === "DISPATCHING" && isSuspectedStuck && (
          <div className="flex flex-col gap-1">
            <span className={cn(textRole.tableCellPrimary, getStatusToneTextClass("warning"))}>
              {t("result.suspectedStuck", { minutes: 5 })}
            </span>
          </div>
        )}
        {occurrence.status === "PENDING" && (
          <span className={textRole.tableCellSecondary}>{t("status.PENDING")}</span>
        )}
      </TableCell>

      <TableCell className={cn(textRole.tableCellSecondary, "py-2.5 text-right font-mono")}>
        {occurrence.durationMs !== null ? `${occurrence.durationMs}ms` : "-"}
      </TableCell>
    </TableRow>
  )
}

function FailureMessage({ message, t }: { message: string | null; t: ReturnType<typeof useTranslations<"scan.scheduled.occurrences">> }) {
  if (!message) {
    return <span className={textRole.compactCaption}>{t("result.noDiagnostics")}</span>
  }
  return (
    <p className={cn(textRole.compactCaption, "break-all font-mono")}>{message}</p>
  )
}
