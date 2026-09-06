"use client"

import React from "react"
import { useTranslations, useLocale } from "next-intl"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { useDetailShellReadySignal } from "@/components/shared/loading/detail-shell-ready-context"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { createAppError } from "@/lib/errors/app-error"
import { normalizeError } from "@/lib/errors/normalize-error"
import { useScanOverviewState } from "@/components/scan/history/scan-overview-state"
import { useScanExecutedEngineDisplay } from "@/hooks/use-scan-executed-engine-display"
import { ScanOverviewLoadingState } from "@/components/scan/history/scan-overview-loading-state"
import {
  SCAN_OVERVIEW_PRIMARY_COLUMN_CLASS,
  SCAN_OVERVIEW_WORKBENCH_CLASS,
} from "@/components/scan/history/scan-overview-layout"
import { ScanOverviewRuntimeDetails } from "@/components/scan/history/scan-overview-sections"
import {
  buildTaskProgressLogsByTaskId,
  toRuntimeTaskItems,
} from "@/components/scan/history/scan-runtime-detail-utils"
import {
  RuntimeHeader,
  RuntimeOverviewSidePanel,
  RuntimeSummarySection,
  RuntimeTaskList,
  hasLiveRuntimeDuration,
  type RuntimeDetailTab,
  useRuntimeDurationNow,
} from "@/components/scan/history/scan-runtime-detail-drawer"

interface ScanOverviewProps { scanId: number }

export function ScanOverview({ scanId }: ScanOverviewProps) {
  const t = useTranslations("scan.history.overview")
  const tStatus = useTranslations("common.status")
  const locale = useLocale()
  const [detailTab, setDetailTab] = React.useState<RuntimeDetailTab>("logs")

  const { scan, isLoading, error, refetch, logs, logsLoading, autoRefresh, setAutoRefresh, isRunning } = useScanOverviewState({
    scanId,
    t,
    logsEnabled: Boolean(scanId),
  })
  const executedEngineDisplay = useScanExecutedEngineDisplay(scan ? [scan] : [])
  const isInitialLoading = (isLoading && !scan) || (Boolean(scan) && executedEngineDisplay.isLoading)
  const detailShellReady = useDetailShellReadySignal(!isInitialLoading)
  const runtimeTasks = React.useMemo(
    () => scan
      ? toRuntimeTaskItems(scan.runtimeTasks ?? [], scan, executedEngineDisplay.engineNamesById, executedEngineDisplay.engineDescriptionsById)
      : [],
    [executedEngineDisplay.engineDescriptionsById, executedEngineDisplay.engineNamesById, scan]
  )
  const runtimeDurationNow = useRuntimeDurationNow(hasLiveRuntimeDuration(scan, runtimeTasks))
  const taskProgressLogsByTaskId = React.useMemo(() => buildTaskProgressLogsByTaskId(logs), [logs])

  if (detailShellReady?.deferInitialSkeleton && isInitialLoading) {
    return null
  }

  const queryError = error ?? executedEngineDisplay.error

  if (!isInitialLoading && queryError) {
    return (
      <AppErrorState
        error={normalizeError(queryError, { notFoundKind: "unexpected-error" })}
        onRetry={() => Promise.all([refetch(), executedEngineDisplay.refetch()])}
        actionHref="/scan/history/"
        variant="section"
      />
    )
  }

  if (!isInitialLoading && !scan) {
    return (
      <AppErrorState
        error={createAppError("resource-not-found")}
        resourceLabel={locale === "zh" ? "扫描" : "Scan"}
        actionHref="/scan/history/"
        variant="section"
      />
    )
  }

  return (
    <ContentHandoff
      owner="scan-overview-content"
      isLoading={isInitialLoading}
      skeleton={<ScanOverviewLoadingState />}
      className="flex min-h-0 flex-1 flex-col"
      skeletonClassName="flex min-h-0 flex-1 flex-col"
      contentClassName="flex min-h-0 flex-1 flex-col"
    >
      {scan ? (
        <div className={SCAN_OVERVIEW_WORKBENCH_CLASS}>
          <div className={SCAN_OVERVIEW_PRIMARY_COLUMN_CLASS}>
            <RuntimeHeader scan={scan} tasks={runtimeTasks} now={runtimeDurationNow} locale={locale} t={t} statusLabel={tStatus} />
            <RuntimeSummarySection scan={scan} tasks={runtimeTasks} t={t} />
            <RuntimeTaskList tasks={runtimeTasks} taskProgressLogsByTaskId={taskProgressLogsByTaskId} now={runtimeDurationNow} t={t} statusLabel={tStatus} />
            <ScanOverviewRuntimeDetails
              t={t}
              scan={scan}
              detailTab={detailTab}
              setDetailTab={setDetailTab}
              logs={logs}
              logsLoading={logsLoading}
              isRunning={isRunning}
              autoRefresh={autoRefresh}
              setAutoRefresh={setAutoRefresh}
            />
          </div>
          <RuntimeOverviewSidePanel
            scan={scan}
            executedEngineNames={executedEngineDisplay.engineNamesByScanId.get(scan.id) ?? []}
            locale={locale}
            t={t}
            statusLabel={tStatus}
          />
        </div>
      ) : null}
    </ContentHandoff>
  )
}
