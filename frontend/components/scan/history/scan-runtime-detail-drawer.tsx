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
  CheckCircle2,
  Loader2,
  Circle,
  ChevronDown
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

function statusIcon(status: RuntimeTaskItem["status"]) {
  switch (status) {
    case "completed":
      return <CheckCircle2 className="h-5 w-5 text-emerald-500" />
    case "failed":
      return <AlertTriangle className="h-5 w-5 text-destructive" />
    case "running":
      return <Loader2 className="h-5 w-5 text-amber-500 animate-spin" />
    default:
      return <Circle className="h-5 w-5 text-muted-foreground/30" />
  }
}

function RuntimeHeader({ scan, locale, t }: { scan: ScanRecord; locale: string; t: TFn }) {
  return (
    <div className="flex flex-col gap-6 xl:flex-row xl:items-start xl:justify-between py-2">
      <div className="min-w-0 space-y-4">
        {/* Domain as Hero Text */}
        <div className="flex items-center gap-3 flex-wrap">
          <h2 className="text-3xl font-bold tracking-tight break-all">
            {scan.target?.name || t("runtimeDrawer.emptyTarget")}
          </h2>
          <Badge
            variant="secondary"
            className={cn(
              "text-sm px-2.5 py-0.5",
              scan.status === "completed" && "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 hover:bg-emerald-500/25",
              scan.status === "running" && "bg-amber-500/15 text-amber-600 dark:text-amber-400 hover:bg-amber-500/25",
              scan.status === "failed" && "bg-destructive/15 text-destructive hover:bg-destructive/25"
            )}
          >
            {scan.status}
          </Badge>
          <Badge variant="outline" className="text-muted-foreground">#{scan.id}</Badge>
          {scan.failure?.kind ? (
            <Badge variant="outline" className="border-destructive/30 bg-destructive/5 text-destructive">
              {scan.failure.kind}
            </Badge>
          ) : null}
        </div>

        {/* Inline metadata without boxes */}
        <div className="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm text-muted-foreground">
          <div className="flex items-center gap-1.5">
            <span className="font-medium text-foreground">{t("runtimeDrawer.meta.createdAt")}:</span>
            <span>{formatDateTime(scan.createdAt, locale)}</span>
          </div>
          {scan.stoppedAt && (
            <div className="flex items-center gap-1.5">
              <span className="font-medium text-foreground">{t("runtimeDrawer.meta.stoppedAt")}:</span>
              <span>{formatDateTime(scan.stoppedAt, locale)}</span>
            </div>
          )}
          <div className="flex items-center gap-1.5">
            <span className="font-medium text-foreground text-xs uppercase tracking-wider bg-muted/50 px-2 py-0.5 rounded-md border border-border/40">
              {scan.workflowNames?.join(", ") || "-"}
            </span>
          </div>
        </div>

        {scan.status === "failed" ? (
          <div className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 max-w-2xl">
            <div className="flex items-center gap-2 text-sm font-medium text-foreground">
              <AlertTriangle className="h-4 w-4 text-destructive" />
              {t("runtimeDrawer.failure.title")}
            </div>
            <p className="mt-1 text-sm text-muted-foreground break-words">{scan.failure?.message || scan.errorMessage || t("runtimeDrawer.failure.empty")}</p>
          </div>
        ) : null}
      </div>
    </div>
  )
}

function RuntimeStatusSummary({ scan, tasks, t }: { scan: ScanRecord; tasks: RuntimeTaskItem[]; t: TFn }) {
  const completedCount = tasks.filter((item) => item.status === "completed").length
  const failedCount = tasks.filter((item) => item.status === "failed").length
  const runningCount = tasks.filter((item) => item.status === "running").length

  return (
    <div className="space-y-3 pt-2">
      <div className="flex items-center justify-between text-sm">
        <div className="flex items-center gap-4">
          <span className="font-medium flex items-center gap-1.5">
            {scan.progress || 0}% <span className="text-muted-foreground font-normal">{t("runtimeDrawer.meta.progress")}</span>
          </span>
        </div>
        <div className="flex items-center gap-3 text-xs">
          {completedCount > 0 && <span className="flex items-center gap-1 text-emerald-600 dark:text-emerald-400"><CheckCircle2 className="h-3.5 w-3.5" /> {completedCount} {t("runtimeDrawer.statusSummary.completed")}</span>}
          {runningCount > 0 && <span className="flex items-center gap-1 text-amber-600 dark:text-amber-500"><Loader2 className="h-3.5 w-3.5 animate-spin" /> {runningCount} {t("runtimeDrawer.statusSummary.running")}</span>}
          {failedCount > 0 && <span className="flex items-center gap-1 text-destructive"><AlertTriangle className="h-3.5 w-3.5" /> {failedCount} {t("runtimeDrawer.statusSummary.failed")}</span>}
        </div>
      </div>
      <Progress
        value={scan.progress || 0}
        className="h-2"
        indicatorClassName={cn(
          scan.status === "failed" ? "bg-destructive" : scan.status === "completed" ? "bg-emerald-500" : "bg-amber-500"
        )}
      />
    </div>
  )
}

function RuntimeTaskList({ tasks, t }: { tasks: RuntimeTaskItem[]; t: TFn }) {
  if (tasks.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-border/60 bg-muted/20 p-6 text-sm text-muted-foreground mt-8">
        {t("runtimeDrawer.tasks.empty")}
      </div>
    )
  }

  return (
    <div className="mt-8">
      <h3 className="text-sm font-semibold mb-6 uppercase tracking-wider text-muted-foreground">
        {t("runtimeDrawer.tasks.title")}
      </h3>

      {/* Vertical Timeline container */}
      <div className="relative pl-6 before:absolute before:inset-y-2 before:left-[11px] before:w-px before:bg-border">
        {tasks.map((task) => (
          <Collapsible key={task.id} className="relative mb-8 last:mb-0 group/task">
            {/* Timeline icon */}
            <div className="absolute -left-[30px] top-1 flex h-6 w-6 items-center justify-center bg-background rounded-full border border-border shadow-sm z-10 transition-transform group-hover/task:scale-110">
              {statusIcon(task.status)}
            </div>

            <CollapsibleTrigger className="flex w-full items-start justify-between gap-4 text-left p-1 rounded-md outline-none focus-visible:ring-2 ring-ring ring-offset-2">
              <div className="flex min-w-0 flex-1 items-start justify-between gap-4 text-left">
                <div className="min-w-0 space-y-1.5">
                  <div className="flex items-center gap-2 flex-wrap">
                    <p className="text-base font-semibold text-foreground">{task.title}</p>
                    <Badge variant="outline" className="text-[10px] h-5 px-1.5 uppercase font-medium">{task.subtitle}</Badge>
                    {task.status === "running" && <span className="flex h-2 w-2 rounded-full bg-amber-500 animate-pulse ml-2" />}
                    {task.failureKind ? (
                      <Badge variant="outline" className="border-destructive/30 bg-destructive/10 text-destructive text-xs">
                        {task.failureKind}
                      </Badge>
                    ) : null}
                  </div>
                  <p className="text-sm text-muted-foreground break-words font-medium opacity-80">
                    {task.detail || (task.status === "running" ? "Loading..." : t("runtimeDrawer.tasks.noDetail"))}
                  </p>
                </div>
                <div className="shrink-0 text-right text-xs text-muted-foreground/80 mt-1 mr-2 opacity-0 group-hover/task:opacity-100 transition-opacity">
                  <div className="flex items-center justify-end gap-1.5 bg-muted/40 px-2 py-1 rounded-md border border-border/40">
                    <Clock className="h-3.5 w-3.5" />
                    <span>{formatDuration(task.duration)}</span>
                  </div>
                </div>
              </div>
            </CollapsibleTrigger>
            <CollapsibleContent className="pt-3 pr-2">
              <div className="rounded-lg border border-border/50 bg-muted/10 p-4 shadow-sm">
                <div className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                  <IconTerminal className="h-3.5 w-3.5" />
                  {t("runtimeDrawer.tasks.logTitle")}
                </div>
                <div className="space-y-[1px] font-mono text-xs">
                  {task.logLines.map((line, index) => (
                    <div key={`${task.id}-log-${index}`} className="text-muted-foreground/80 break-words py-0.5 hover:text-foreground hover:bg-muted/30 px-1 rounded transition-colors">
                      {line}
                    </div>
                  ))}
                </div>
              </div>
            </CollapsibleContent>
          </Collapsible>
        ))}
      </div>
    </div>
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
    <div className="mt-8">
      <h3 className="text-sm font-semibold mb-4 uppercase tracking-wider text-muted-foreground">
        {t("runtimeDrawer.assets.title")}
      </h3>
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {items.map((item) => (
          <div key={item.label} className="rounded-lg border border-border/50 bg-muted/20 px-4 py-3 hover:bg-muted/30 transition-colors">
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <item.icon className="h-3.5 w-3.5" />
              <span>{item.label}</span>
            </div>
            <p className="mt-2 text-lg font-semibold">{item.value}</p>
          </div>
        ))}
      </div>
    </div>
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
              <div className="space-y-10">
                <div className="space-y-8">
                  <RuntimeHeader scan={runtimeScan} locale={locale} t={t} />
                  <RuntimeStatusSummary scan={runtimeScan} tasks={runtimeTasks} t={t} />
                </div>
                <Separator />

                <RuntimeTaskList tasks={runtimeTasks} t={t} />

                <Separator className="my-8 opacity-50" />

                {/* Configuration Panel */}
                <div>
                  <h3 className="text-sm font-semibold mb-4 uppercase tracking-wider text-muted-foreground">
                    {t("runtimeDrawer.config.title")}
                  </h3>
                  <div className="rounded-lg border border-border/50 bg-black/5 dark:bg-black/20 p-4 shadow-sm">
                    <pre className="overflow-x-auto font-mono text-xs leading-6 text-muted-foreground/90">{JSON.stringify(runtimeScan.configuration || {}, null, 2)}</pre>
                  </div>
                </div>

                {/* Scan Logs Panel */}
                <div>
                  <h3 className="text-sm font-semibold mb-4 uppercase tracking-wider text-muted-foreground">
                    {t("runtimeDrawer.scanLogs.title")}
                  </h3>
                  <div className="min-h-[420px] rounded-lg border border-border/50 shadow-sm overflow-hidden bg-background/50">
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
                  </div>
                </div>

                <RuntimeAssetsSummary scan={runtimeScan} t={t} />

                <Separator className="my-8 opacity-50" />

                {/* Debug Panel */}
                <Collapsible className="group/debug">
                  <CollapsibleTrigger className="flex w-full items-center justify-between outline-none">
                    <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground group-hover/debug:text-foreground transition-colors flex items-center gap-2">
                      {t("runtimeDrawer.debug.title")}
                      <ChevronDown className="h-4 w-4 text-muted-foreground/50 transition-transform group-data-[state=open]/debug:rotate-180" />
                    </h3>
                  </CollapsibleTrigger>
                  <CollapsibleContent className="pt-4">
                    <div className="space-y-3 text-xs text-muted-foreground">
                      <p>{t("runtimeDrawer.debug.note")}</p>
                      <pre className="overflow-x-auto rounded-lg border border-border/50 bg-black/5 dark:bg-black/20 p-4 shadow-sm font-mono text-[11px] leading-5">{JSON.stringify({
                        id: runtimeScan.id,
                        status: runtimeScan.status,
                        failure: runtimeScan.failure,
                        tasks: runtimeTasks,
                      }, null, 2)}</pre>
                    </div>
                  </CollapsibleContent>
                </Collapsible>
              </div>
            )}
          </div>
        </div>
      </SheetContent>
    </Sheet>
  )
}
