"use client"

import React from "react"
import { useTranslations, useLocale } from "next-intl"
import { AlertTriangle } from "@/components/icons"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { useScanOverviewState } from "@/components/scan/history/scan-overview-state"
import { ScanLogsPanel, ScanOverviewAssets, ScanOverviewHeader, ScanStageProgress, ScanVulnerabilitySummary } from "@/components/scan/history/scan-overview-sections"

interface ScanOverviewProps { scanId: number }

export function ScanOverview({ scanId }: ScanOverviewProps) {
  const t = useTranslations("scan.history.overview")
  const tStatus = useTranslations("scan.history.status")
  const tProgress = useTranslations("scan.progress")
  const locale = useLocale()

  const { scan, isLoading, error, logs, logsLoading, activeTab, setActiveTab, autoRefresh, setAutoRefresh, isRunning, vulnSummary, startedAt, completedAt, assetCards } = useScanOverviewState({ scanId, t })

  if (isLoading) {
    return <div className="space-y-6"><div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">{[...Array(6)].map((_, i) => <Card key={i}><CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2"><Skeleton className="h-4 w-24" /><Skeleton className="h-4 w-4" /></CardHeader><CardContent><Skeleton className="h-8 w-16" /></CardContent></Card>)}</div></div>
  }

  if (error || !scan) {
    return <div className="flex flex-col items-center justify-center py-12"><AlertTriangle className="h-10 w-10 text-destructive mb-4" /><p className="text-muted-foreground">{t("loadError")}</p></div>
  }

  return (
    <div className="flex flex-col gap-6 flex-1 min-h-0">
      <ScanOverviewHeader t={t} tStatus={tStatus} locale={locale} scan={scan} startedAt={startedAt} completedAt={completedAt} />
      <ScanOverviewAssets t={t} assetCards={assetCards} />
      <div className="grid gap-4 md:grid-cols-[280px_1fr] flex-1 min-h-0">
        <div className="flex flex-col gap-4 min-h-0">
          <ScanStageProgress t={t} tProgress={tProgress} stageProgress={scan.stageProgress} />
          <ScanVulnerabilitySummary t={t} scanId={scanId} vulnSummary={vulnSummary} />
        </div>
        <ScanLogsPanel t={t} activeTab={activeTab} setActiveTab={setActiveTab} isRunning={isRunning} autoRefresh={autoRefresh} setAutoRefresh={setAutoRefresh} logs={logs} logsLoading={logsLoading} configuration={scan.configuration} />
      </div>
    </div>
  )
}
