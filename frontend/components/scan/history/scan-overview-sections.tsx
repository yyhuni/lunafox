import React from "react"
import Link from "next/link"
import dynamic from "next/dynamic"
import {
  Calendar,
  ChevronRight,
  Cpu,
  HardDrive,
  Clock,
} from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ScanLogList } from "@/components/scan/scan-log-list"
import { cn } from "@/lib/utils"
import type { ScanRecord, StageProgressItem, StageStatus } from "@/types/scan.types"
import type { ScanOverviewState } from "@/components/scan/history/scan-overview-state"
import {
  formatDate,
  formatDuration,
  formatStageDuration,
  getStatusIcon,
  SCAN_STATUS_STYLES,
  STAGE_STATUS_PRIORITY,
} from "@/components/scan/history/scan-overview-utils"
import { IconCircleCheck, IconCircleX, IconClock } from "@/components/icons"

const YamlEditor = dynamic(
  () => import("@/components/ui/yaml-editor").then((mod) => ({ default: mod.YamlEditor })),
  {
    loading: () => (
      <div className="flex items-center justify-center h-full text-muted-foreground text-sm">加载编辑器中...</div>
    ),
    ssr: false,
  }
)

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

function PulsingDot({ className }: { className?: string }) {
  return (
    <span className={cn("relative flex h-3 w-3", className)}>
      <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-current opacity-75" />
      <span className="relative inline-flex h-3 w-3 rounded-full bg-current" />
    </span>
  )
}

function StageStatusIcon({ status }: { status: StageStatus }) {
  switch (status) {
    case "completed":
      return <IconCircleCheck className="h-5 w-5 text-[var(--success)]" />
    case "running":
      return <PulsingDot className="text-[var(--warning)]" />
    case "failed":
      return <IconCircleX className="h-5 w-5 text-[var(--error)]" />
    case "cancelled":
      return <IconCircleX className="h-5 w-5 text-muted-foreground" />
    default:
      return <IconClock className="h-5 w-5 text-muted-foreground" />
  }
}

interface ScanOverviewHeaderProps {
  t: TranslationFn
  tStatus: TranslationFn
  locale: string
  scan: ScanRecord
  startedAt?: string
  completedAt?: string
}

export function ScanOverviewHeader({
  t,
  tStatus,
  locale,
  scan,
  startedAt,
  completedAt,
}: ScanOverviewHeaderProps) {
  const statusIconConfig = React.useMemo(() => getStatusIcon(scan.status ?? "pending"), [scan.status])
  const StatusIcon = statusIconConfig.icon
  const statusStyle = SCAN_STATUS_STYLES[scan.status ?? "pending"] || "bg-muted text-muted-foreground"

  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-6 text-sm text-muted-foreground">
        <div className="flex items-center gap-1.5">
          <Calendar className="h-4 w-4" />
          <span>{t("startedAt")}: {formatDate(startedAt, locale)}</span>
        </div>
        <div className="flex items-center gap-1.5">
          <Clock className="h-4 w-4" />
          <span>{t("duration")}: {formatDuration(startedAt, completedAt)}</span>
        </div>
        {scan.workflowNames && scan.workflowNames.length > 0 && (
          <div className="flex items-center gap-1.5">
            <Cpu className="h-4 w-4" />
            <span>{scan.workflowNames.join(", ")}</span>
          </div>
        )}
        {scan.workerName && (
          <div className="flex items-center gap-1.5">
            <HardDrive className="h-4 w-4" />
            <span>{scan.workerName}</span>
          </div>
        )}
      </div>
      <Badge variant="outline" className={statusStyle}>
        <StatusIcon className={`h-3.5 w-3.5 mr-1.5 ${statusIconConfig.animate ? "animate-spin" : ""}`} />
        {tStatus(scan.status)}
      </Badge>
    </div>
  )
}

interface ScanOverviewAssetsProps {
  t: TranslationFn
  assetCards: ScanOverviewState["assetCards"]
}

export function ScanOverviewAssets({ t, assetCards }: ScanOverviewAssetsProps) {
  return (
    <div>
      <h3 className="text-lg font-semibold mb-4">{t("assetsTitle")}</h3>
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-6">
        {assetCards.map((card) => (
          <Link key={card.title} href={card.href} className="block">
            <div
              className="group relative p-4 hover:bg-accent/5 transition-[background-color,border-color,box-shadow] duration-300 cursor-pointer"
              style={{ background: "var(--card)" }}
            >
              <div className="absolute inset-0 border border-border/40 group-hover:border-primary/30 transition-colors" />
              <div className="absolute top-0 right-0 h-2 w-2 border-r border-t border-primary/50" />
              <div className="absolute bottom-0 left-0 h-2 w-2 border-l border-b border-primary/50" />

              <div className="relative z-10">
                <div className="flex justify-between items-start mb-2">
                  <div className="text-[10px] font-mono text-muted-foreground bg-muted px-1.5 py-0.5 rounded-sm">
                    {card.code}
                  </div>
                  <card.icon className="h-4 w-4 text-muted-foreground/70 group-hover:text-primary transition-colors" />
                </div>

                <div className="text-3xl font-light tracking-tight text-foreground group-hover:translate-x-1 transition-transform duration-300">
                  {card.value.toLocaleString()}
                </div>

                <div className="mt-2 flex items-center gap-2">
                  <div className="h-px flex-1 bg-border border-t border-dashed border-muted-foreground/20" />
                  <span className="text-[11px] text-foreground/85 font-mono uppercase tracking-wider">
                    {card.title}
                  </span>
                </div>
              </div>
            </div>
          </Link>
        ))}
      </div>
    </div>
  )
}

interface ScanStageProgressProps {
  t: TranslationFn
  tProgress: TranslationFn
  stageProgress?: Record<string, StageProgressItem>
}

export function ScanStageProgress({ t, tProgress, stageProgress }: ScanStageProgressProps) {
  return (
    <Card className="flex-1 min-h-0">
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-3">
        <CardTitle className="text-sm font-medium">{t("stagesTitle")}</CardTitle>
        {stageProgress && (
          <span className="text-xs text-muted-foreground">
            {Object.values(stageProgress).filter((progress) => progress.status === "completed").length}/
            {Object.keys(stageProgress).length} {t("stagesCompleted")}
          </span>
        )}
      </CardHeader>
      <CardContent className="pt-0 flex flex-col flex-1 min-h-0">
        {stageProgress && Object.keys(stageProgress).length > 0 ? (
          <div className="space-y-1 flex-1 min-h-0 overflow-y-auto pr-1">
            {(Object.entries(stageProgress) as Array<[string, StageProgressItem]>)
              .toSorted(([, a], [, b]) => {
                const priorityA = STAGE_STATUS_PRIORITY[a.status as StageStatus] ?? 99
                const priorityB = STAGE_STATUS_PRIORITY[b.status as StageStatus] ?? 99
                if (priorityA !== priorityB) {
                  return priorityA - priorityB
                }
                return (a.order ?? 0) - (b.order ?? 0)
              })
              .map(([stageName, stageProgressItem]) => {
                const isRunning = stageProgressItem.status === "running"
                return (
                  <div
                    key={stageName}
                    className={cn(
                      "flex items-center justify-between py-2 px-2 rounded-md transition-colors text-sm",
                      isRunning && "bg-[var(--warning)]/10 border border-[var(--warning)]/30",
                      stageProgressItem.status === "completed" && "text-muted-foreground",
                      stageProgressItem.status === "failed" && "bg-[var(--error)]/10 text-[var(--error)]",
                      stageProgressItem.status === "cancelled" && "text-muted-foreground"
                    )}
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <StageStatusIcon status={stageProgressItem.status} />
                      <span className={cn("truncate", isRunning && "font-medium text-foreground")}>
                        {tProgress(`stages.${stageName}`)}
                      </span>
                    </div>
                    <span className="text-xs text-muted-foreground font-mono shrink-0 ml-2">
                      {stageProgressItem.status === "completed" && stageProgressItem.duration
                        ? formatStageDuration(stageProgressItem.duration)
                        : stageProgressItem.status === "running"
                          ? tProgress("stage_running")
                          : stageProgressItem.status === "pending"
                            ? "--"
                            : ""}
                    </span>
                  </div>
                )
              })}
          </div>
        ) : (
          <div className="text-sm text-muted-foreground text-center py-4">
            {t("noStages")}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

interface ScanVulnerabilitySummaryProps {
  t: TranslationFn
  scanId: number
  vulnSummary: ScanOverviewState["vulnSummary"]
}

export function ScanVulnerabilitySummary({ t, scanId, vulnSummary }: ScanVulnerabilitySummaryProps) {
  return (
    <Link href={`/scan/history/${scanId}/vulnerabilities/`} className="block">
      <Card className="hover:border-primary/50 transition-colors cursor-pointer">
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">{t("vulnerabilitiesTitle")}</CardTitle>
          <ChevronRight className="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent className="pt-0">
          <div className="flex items-center gap-3 flex-wrap">
            <div className="flex items-center gap-1.5">
              <div className="w-2.5 h-2.5 rounded-full bg-[var(--error)]" />
              <span className="text-sm font-medium">{vulnSummary.critical}</span>
            </div>
            <div className="flex items-center gap-1.5">
              <div className="w-2.5 h-2.5 rounded-full bg-[var(--error)]/70" />
              <span className="text-sm font-medium">{vulnSummary.high}</span>
            </div>
            <div className="flex items-center gap-1.5">
              <div className="w-2.5 h-2.5 rounded-full bg-[var(--warning)]" />
              <span className="text-sm font-medium">{vulnSummary.medium}</span>
            </div>
            <div className="flex items-center gap-1.5">
              <div className="w-2.5 h-2.5 rounded-full bg-[var(--info)]" />
              <span className="text-sm font-medium">{vulnSummary.low}</span>
            </div>
            <span className="text-xs text-muted-foreground ml-auto">
              {t("totalVulns", { count: vulnSummary.total ?? 0 })}
            </span>
          </div>
        </CardContent>
      </Card>
    </Link>
  )
}

interface ScanLogsPanelProps {
  t: TranslationFn
  activeTab: "logs" | "config"
  setActiveTab: (value: "logs" | "config") => void
  isRunning: boolean
  autoRefresh: boolean
  setAutoRefresh: (value: boolean) => void
  logs: ScanOverviewState["logs"]
  logsLoading: boolean
  yamlConfiguration?: string | null
}

export function ScanLogsPanel({
  t,
  activeTab,
  setActiveTab,
  isRunning,
  autoRefresh,
  setAutoRefresh,
  logs,
  logsLoading,
  yamlConfiguration,
}: ScanLogsPanelProps) {
  return (
    <div className="flex flex-col min-h-0 rounded-lg overflow-hidden border">
      <div className="flex items-center justify-between px-3 py-2 bg-muted/30 border-b shrink-0">
        <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as "logs" | "config")}>
          <TabsList variant="minimal-tab">
            <TabsTrigger variant="minimal-tab" value="logs">
              {t("logsTitle")}
            </TabsTrigger>
            <TabsTrigger variant="minimal-tab" value="config">
              {t("configTitle")}
            </TabsTrigger>
          </TabsList>
        </Tabs>
        {activeTab === "logs" && isRunning && (
          <div className="flex items-center gap-2">
            <Switch
              id="log-auto-refresh"
              checked={autoRefresh}
              onCheckedChange={setAutoRefresh}
              className="scale-75"
            />
            <Label htmlFor="log-auto-refresh" className="text-xs cursor-pointer">
              {t("autoRefresh")}
            </Label>
          </div>
        )}
      </div>

      <div className="flex-1 min-h-0">
        {activeTab === "logs" ? (
          <ScanLogList logs={logs} loading={logsLoading} />
        ) : (
          <div className="h-full">
            {yamlConfiguration ? (
              <YamlEditor
                value={yamlConfiguration}
                onChange={() => {}}
                disabled={true}
                className="h-full"
              />
            ) : (
              <div className="flex items-center justify-center h-full text-muted-foreground text-sm">
                {t("noConfig")}
              </div>
            )}
          </div>
        )}
      </div>

      {activeTab === "logs" && (
        <div className="flex items-center px-4 py-2 bg-muted/50 border-t text-xs text-muted-foreground shrink-0">
          <span>{logs.length} 条记录</span>
          {isRunning && autoRefresh && (
            <>
              <Separator orientation="vertical" className="h-3 mx-3" />
              <span className="flex items-center gap-1.5">
                <span className="size-1.5 rounded-full bg-[var(--success)] animate-pulse" />
                每 3 秒刷新
              </span>
            </>
          )}
        </div>
      )}
    </div>
  )
}
