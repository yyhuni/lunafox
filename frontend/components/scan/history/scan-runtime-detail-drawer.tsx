"use client"

import React from "react"
import { useLocale, useTranslations } from "next-intl"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import {
  AlertTriangle,
  Clock,
  HardDrive,
  IconTerminal,
  Link2,
  Network,
  Server,
} from "@/components/icons"
import { ScanLogsPanel } from "@/components/scan/history/scan-overview-sections"
import { useScanOverviewState } from "@/components/scan/history/scan-overview-state"
import type { ScanRecord, StageProgressItem } from "@/types/scan.types"
import { cn } from "@/lib/utils"

type TFn = ReturnType<typeof useTranslations>

interface ScanRuntimeDetailDrawerProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  scan: ScanRecord | null
}

interface RuntimeTaskItem {
  id: string
  title: string
  subtitle: string
  status: NonNullable<StageProgressItem["status"]>
  failureKind?: string
  detail?: string
  logLines: string[]
  duration?: number
  order: number
}

function formatDateTime(value?: string, locale: string = "zh") {
  if (!value) return "-"
  try {
    return new Date(value).toLocaleString(locale === "zh" ? "zh-CN" : "en-US", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
    })
  } catch {
    return value
  }
}

function formatDuration(duration?: number) {
  if (duration === undefined || duration === null) return "-"
  if (duration < 1) return "<1s"
  if (duration < 60) return `${Math.round(duration)}s`
  const minutes = Math.floor(duration / 60)
  const seconds = Math.round(duration % 60)
  return seconds > 0 ? `${minutes}m ${seconds}s` : `${minutes}m`
}

function statusTone(status: RuntimeTaskItem["status"]) {
  switch (status) {
    case "failed":
      return "border-destructive/30 bg-destructive/5"
    case "running":
      return "border-amber-500/30 bg-amber-500/5"
    case "completed":
      return "border-emerald-500/25 bg-emerald-500/5"
    case "cancelled":
      return "border-border/60 bg-muted/20"
    default:
      return "border-border/60 bg-card"
  }
}

function buildTaskLogs(item: StageProgressItem, scan: ScanRecord | null, isCanonicalFailure: boolean): string[] {
  const lines: string[] = []
  if (item.detail) lines.push(item.detail)
  if (item.error) lines.push(item.error)
  if (item.reason) lines.push(item.reason)
  if (isCanonicalFailure && scan?.failure?.message) lines.push(scan.failure.message)
  if (lines.length === 0) {
    lines.push(scan?.status === "failed" && isCanonicalFailure ? scan.failure?.message || scan.errorMessage || "No task logs available." : "No task logs available.")
  }
  return lines
}

function buildRuntimeTasks(scan: ScanRecord | null): RuntimeTaskItem[] {
  if (!scan?.stageProgress) {
    return (scan?.workflowNames || []).map((workflowName, index) => ({
      id: `${scan?.id ?? "scan"}-workflow-${index}`,
      title: workflowName,
      subtitle: `Task #${index + 1}`,
      status: scan?.status === "failed" ? "failed" : scan?.status === "completed" ? "completed" : scan?.status === "running" ? "running" : "pending",
      failureKind: scan?.failure?.kind,
      detail: scan?.failure?.message || scan?.errorMessage,
      logLines: scan?.failure?.message ? [scan.failure.message] : [scan?.errorMessage || "No task logs available."],
      order: index,
    }))
  }

  const firstFailedStage = Object.entries(scan.stageProgress).find(([, item]) => item.status === "failed")?.[0]

  return Object.entries(scan.stageProgress)
    .map(([stageName, item], index) => {
      const isCanonicalFailure = stageName === firstFailedStage
      return {
        id: `${scan.id}-${stageName}`,
        title: stageName,
        subtitle: `Stage ${typeof item.order === "number" ? item.order + 1 : index + 1}`,
        status: item.status,
        failureKind: isCanonicalFailure ? scan.failure?.kind : undefined,
        detail: item.error || item.reason || item.detail || (isCanonicalFailure ? scan.failure?.message || scan.errorMessage : undefined),
        logLines: buildTaskLogs(item, scan, isCanonicalFailure),
        duration: item.duration,
        order: item.order ?? index,
      }
    })
    .sort((a, b) => a.order - b.order)
}

function RuntimeHeader({ scan, locale, t }: { scan: ScanRecord; locale: string; t: TFn }) {
  return (
    <Card className="border-border/60">
      <CardContent className="p-5">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <div className="min-w-0 space-y-2">
            <div className="flex items-center gap-2 flex-wrap">
              <Badge variant="secondary">{scan.status}</Badge>
              <Badge variant="outline">#{scan.id}</Badge>
              {scan.failure?.kind ? (
                <Badge variant="outline" className="border-destructive/30 bg-background/80 text-destructive">
                  {scan.failure.kind}
                </Badge>
              ) : null}
            </div>
            <div>
              <h3 className="text-lg font-semibold break-all">{scan.target?.name || t("runtimeDrawer.emptyTarget")}</h3>
              <p className="text-sm text-muted-foreground break-words">{scan.workflowNames?.join(", ") || "-"}</p>
            </div>
            {scan.status === "failed" ? (
              <div className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3">
                <p className="text-sm font-medium text-foreground">{t("runtimeDrawer.failure.title")}</p>
                <p className="mt-1 text-sm text-muted-foreground break-words">{scan.failure?.message || scan.errorMessage || t("runtimeDrawer.failure.empty")}</p>
              </div>
            ) : null}
          </div>
          <div className="grid gap-3 sm:grid-cols-2 xl:min-w-[340px]">
            <div className="rounded-lg border border-border/60 bg-muted/20 p-3">
              <p className="text-xs text-muted-foreground">{t("runtimeDrawer.meta.createdAt")}</p>
              <p className="mt-1 text-sm font-medium">{formatDateTime(scan.createdAt, locale)}</p>
            </div>
            <div className="rounded-lg border border-border/60 bg-muted/20 p-3">
              <p className="text-xs text-muted-foreground">{t("runtimeDrawer.meta.stoppedAt")}</p>
              <p className="mt-1 text-sm font-medium">{formatDateTime(scan.stoppedAt, locale)}</p>
            </div>
            <div className="rounded-lg border border-border/60 bg-muted/20 p-3">
              <p className="text-xs text-muted-foreground">{t("runtimeDrawer.meta.worker")}</p>
              <p className="mt-1 text-sm font-medium">{scan.workerName || "-"}</p>
            </div>
            <div className="rounded-lg border border-border/60 bg-muted/20 p-3">
              <p className="text-xs text-muted-foreground">{t("runtimeDrawer.meta.currentStage")}</p>
              <p className="mt-1 text-sm font-medium break-words">{scan.currentStage || "-"}</p>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function RuntimeStatusSummary({ scan, tasks, t }: { scan: ScanRecord; tasks: RuntimeTaskItem[]; t: TFn }) {
  const completedCount = tasks.filter((item) => item.status === "completed").length
  const failedCount = tasks.filter((item) => item.status === "failed").length
  const runningCount = tasks.filter((item) => item.status === "running").length

  return (
    <Card className="border-border/60">
      <CardHeader className="pb-3">
        <CardTitle className="text-base">{t("runtimeDrawer.statusSummary.title")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">{t("runtimeDrawer.meta.progress")}</span>
            <span className="font-medium">{scan.progress || 0}%</span>
          </div>
          <Progress value={scan.progress || 0} className="h-2" />
        </div>
        <div className="grid gap-3 sm:grid-cols-3">
          <div className="rounded-lg border border-border/60 bg-muted/20 p-3">
            <p className="text-xs text-muted-foreground">{t("runtimeDrawer.statusSummary.completed")}</p>
            <p className="mt-1 text-lg font-semibold">{completedCount}</p>
          </div>
          <div className="rounded-lg border border-border/60 bg-muted/20 p-3">
            <p className="text-xs text-muted-foreground">{t("runtimeDrawer.statusSummary.running")}</p>
            <p className="mt-1 text-lg font-semibold">{runningCount}</p>
          </div>
          <div className="rounded-lg border border-border/60 bg-muted/20 p-3">
            <p className="text-xs text-muted-foreground">{t("runtimeDrawer.statusSummary.failed")}</p>
            <p className="mt-1 text-lg font-semibold">{failedCount}</p>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function RuntimeTaskList({ tasks, t }: { tasks: RuntimeTaskItem[]; t: TFn }) {
  return (
    <Card className="border-border/60">
      <CardHeader className="pb-3">
        <CardTitle className="text-base">{t("runtimeDrawer.tasks.title")}</CardTitle>
        <p className="text-sm text-muted-foreground">{t("runtimeDrawer.tasks.description")}</p>
      </CardHeader>
      <CardContent>
        {tasks.length === 0 ? (
          <div className="rounded-lg border border-dashed border-border/60 bg-muted/20 p-6 text-sm text-muted-foreground">
            {t("runtimeDrawer.tasks.empty")}
          </div>
        ) : (
          <div className="space-y-3">
            {tasks.map((task) => (
              <Collapsible key={task.id} className={cn("rounded-xl border px-4", statusTone(task.status))}>
                <CollapsibleTrigger className="flex w-full items-start justify-between gap-4 py-4 text-left">
                  <div className="flex min-w-0 flex-1 items-start justify-between gap-4 text-left">
                    <div className="min-w-0 space-y-2">
                      <div className="flex items-center gap-2 flex-wrap">
                        <p className="text-sm font-medium text-foreground break-all">{task.title}</p>
                        <Badge variant="outline" className="text-xs">{task.subtitle}</Badge>
                        <Badge variant="secondary" className="text-xs">{task.status}</Badge>
                        {task.failureKind ? (
                          <Badge variant="outline" className="border-destructive/30 bg-background/80 text-destructive text-xs">
                            {task.failureKind}
                          </Badge>
                        ) : null}
                      </div>
                      <p className="text-sm text-muted-foreground break-words">{task.detail || t("runtimeDrawer.tasks.noDetail")}</p>
                    </div>
                    <div className="shrink-0 text-right text-xs text-muted-foreground">
                      <div className="flex items-center justify-end gap-1">
                        <Clock className="h-3.5 w-3.5" />
                        <span>{formatDuration(task.duration)}</span>
                      </div>
                    </div>
                  </div>
                </CollapsibleTrigger>
                <CollapsibleContent className="pb-4">
                  <div className="rounded-lg border border-border/60 bg-background/80 p-4">
                    <div className="mb-3 flex items-center gap-2 text-sm font-medium text-foreground">
                      <IconTerminal className="h-4 w-4" />
                      {t("runtimeDrawer.tasks.logTitle")}
                    </div>
                    <div className="space-y-2">
                      {task.logLines.map((line, index) => (
                        <div key={`${task.id}-log-${index}`} className="rounded-md border border-border/50 bg-muted/20 px-3 py-2 text-sm text-muted-foreground break-words">
                          {line}
                        </div>
                      ))}
                    </div>
                  </div>
                </CollapsibleContent>
              </Collapsible>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function RuntimeAssetsSummary({ scan, t }: { scan: ScanRecord; t: TFn }) {
  const items = [
    { label: t("cards.subdomains"), value: scan.cachedStats?.subdomainsCount ?? 0, icon: Network },
    { label: t("cards.ips"), value: scan.cachedStats?.ipsCount ?? 0, icon: Server },
    { label: t("cards.urls"), value: scan.cachedStats?.endpointsCount ?? 0, icon: Link2 },
    { label: t("cards.websites"), value: scan.cachedStats?.websitesCount ?? 0, icon: HardDrive },
  ]

  return (
    <Card className="border-border/60">
      <CardHeader className="pb-3">
        <CardTitle className="text-base">{t("runtimeDrawer.assets.title")}</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {items.map((item) => (
          <div key={item.label} className="rounded-lg border border-border/60 bg-muted/20 px-4 py-3">
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <item.icon className="h-3.5 w-3.5" />
              <span>{item.label}</span>
            </div>
            <p className="mt-2 text-base font-semibold">{item.value}</p>
          </div>
        ))}
      </CardContent>
    </Card>
  )
}

export function ScanRuntimeDetailDrawer({ open, onOpenChange, scan }: ScanRuntimeDetailDrawerProps) {
  const t = useTranslations("scan.history.overview")
  const locale = useLocale()
  const { scan: freshScan, isLoading, error, logs, logsLoading, activeTab, setActiveTab, autoRefresh, setAutoRefresh, isRunning } = useScanOverviewState({ scanId: scan?.id || 0, t })
  const runtimeScan = freshScan || scan
  const runtimeTasks = React.useMemo(() => buildRuntimeTasks(runtimeScan), [runtimeScan])

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-[94vw] sm:max-w-[960px] p-0">
        <div className="flex h-full min-h-0 flex-col bg-background">
          <SheetHeader className="border-b px-6 py-5 text-left">
            <SheetTitle className="text-xl">{t("runtimeDrawer.title")}</SheetTitle>
          </SheetHeader>
          <div className="flex-1 overflow-y-auto px-6 py-6">
            {!runtimeScan && isLoading ? (
              <div className="space-y-4">
                <Skeleton className="h-24 w-full" />
                <Skeleton className="h-36 w-full" />
                <Skeleton className="h-72 w-full" />
              </div>
            ) : error || !runtimeScan ? (
              <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border/60 py-16 text-center">
                <AlertTriangle className="mb-3 h-8 w-8 text-destructive" />
                <p className="text-sm text-muted-foreground">{t("runtimeDrawer.loadError")}</p>
              </div>
            ) : (
              <div className="space-y-6">
                <RuntimeHeader scan={runtimeScan} locale={locale} t={t} />
                <RuntimeStatusSummary scan={runtimeScan} tasks={runtimeTasks} t={t} />
                <RuntimeTaskList tasks={runtimeTasks} t={t} />
                <Card className="border-border/60">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-base">{t("runtimeDrawer.config.title")}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="rounded-lg border border-border/60 bg-muted/20 p-4">
                      <pre className="overflow-x-auto text-xs leading-6 text-muted-foreground">{JSON.stringify(runtimeScan.configuration || {}, null, 2)}</pre>
                    </div>
                  </CardContent>
                </Card>
                <Card className="border-border/60">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-base">{t("runtimeDrawer.scanLogs.title")}</CardTitle>
                  </CardHeader>
                  <CardContent className="min-h-[420px]">
                    <ScanLogsPanel
                      t={t}
                      activeTab={activeTab}
                      setActiveTab={setActiveTab}
                      isRunning={isRunning}
                      autoRefresh={autoRefresh}
                      setAutoRefresh={setAutoRefresh}
                      logs={logs}
                      logsLoading={logsLoading}
                      configuration={runtimeScan.configuration}
                    />
                  </CardContent>
                </Card>
                <RuntimeAssetsSummary scan={runtimeScan} t={t} />
                <Separator />
                <Card className="border-border/60">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-base">{t("runtimeDrawer.debug.title")}</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-2 text-xs text-muted-foreground">
                    <p>{t("runtimeDrawer.debug.note")}</p>
                    <pre className="overflow-x-auto rounded-lg border border-border/60 bg-muted/20 p-3 text-[11px] leading-5">{JSON.stringify({
                      id: runtimeScan.id,
                      status: runtimeScan.status,
                      failure: runtimeScan.failure,
                      tasks: runtimeTasks,
                    }, null, 2)}</pre>
                  </CardContent>
                </Card>
              </div>
            )}
          </div>
        </div>
      </SheetContent>
    </Sheet>
  )
}
