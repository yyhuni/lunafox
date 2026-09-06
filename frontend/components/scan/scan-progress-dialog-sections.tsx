"use client"

import type { ScanProgressData, StageDetail } from "@/components/scan/scan-progress-dialog-types"
import type { StageStatus } from "@/types/scan.types"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  IconClock,
  IconBan,
  semanticIcons,
} from "@/components/icons"
import {
  getScanStatusBadgeVariant,
  getScanStatusClasses,
  getScanStatusSurfaceClass,
  getScanStatusTextClass,
} from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import { ScanLogList } from "@/components/scan/scan-log-list"
import type { ScanLog } from "@/types/scan.types"
import { formatDateTime } from "@/components/scan/scan-progress-dialog-utils"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

function PulsingDot({ className }: { className?: string }) {
  return (
    <span className={cn("relative flex h-3 w-3", className)}>
      <span className="absolute animate-ping bg-current h-full inline-flex opacity-75 rounded-full w-full" />
      <span className="bg-current h-3 inline-flex relative rounded-full w-3" />
    </span>
  )
}

export function ScanStatusIcon({ status }: { status: string }) {
  switch (status) {
    case "running":
      return <PulsingDot className={getScanStatusTextClass(status)} />
    case "succeeded":
      return <semanticIcons.status.success className={cn("h-5 w-5", getScanStatusTextClass(status))} />
    case "skipped":
      return <IconBan className={cn("h-5 w-5", getScanStatusTextClass(status))} />
    case "cancelled":
      return <semanticIcons.status.cancelled className={cn("h-5 w-5", getScanStatusTextClass(status))} />
    case "failed":
      return <semanticIcons.status.failed className={cn("h-5 w-5", getScanStatusTextClass(status))} />
    case "pending":
      return <PulsingDot className={getScanStatusTextClass(status)} />
    default:
      return <PulsingDot className="text-muted-foreground" />
  }
}

export function ScanStatusBadge({ status, t }: { status: string; t: TranslationFn }) {
  const className = getScanStatusClasses(status)
  const classNameVariant = getScanStatusBadgeVariant(status)
  const label = t(`status_${status}`)
  return (
    <Badge variant={classNameVariant} className={classNameVariant === "outline" ? className : undefined}>
      {label}
    </Badge>
  )
}

function StageStatusIcon({ status }: { status: StageStatus }) {
  switch (status) {
    case "succeeded":
      return <semanticIcons.status.success className={cn("h-5 w-5", getScanStatusTextClass(status))} />
    case "running":
      return <PulsingDot className={getScanStatusTextClass(status)} />
    case "failed":
      return <semanticIcons.status.failed className={cn("h-5 w-5", getScanStatusTextClass(status))} />
    case "cancelled":
      return <semanticIcons.status.cancelled className={cn("h-5 w-5", getScanStatusTextClass(status))} />
    case "pending":
      return <IconClock className={cn("h-5 w-5", getScanStatusTextClass(status))} />
    default:
      return <IconClock className="h-5 text-muted-foreground w-5" />
  }
}

function StageRow({ stage, engineName, t }: { stage: StageDetail; engineName: string; t: TranslationFn }) {
  return (
    <div
      className={cn(
        "flex items-center justify-between py-3 px-4 rounded-lg transition-colors",
        stage.status === "running" && getScanStatusSurfaceClass(stage.status),
        stage.status === "succeeded" && "bg-muted/50",
        stage.status === "skipped" && "bg-muted/50",
        stage.status === "failed" && getScanStatusSurfaceClass(stage.status),
        stage.status === "cancelled" && getScanStatusSurfaceClass(stage.status),
      )}
    >
      <div className="flex gap-3 items-center">
        <StageStatusIcon status={stage.status} />
        <div>
          <span className="font-medium">{engineName}</span>
          {stage.detail && (
            <p className="mt-0.5 text-muted-foreground text-xs">
              {stage.detail}
            </p>
          )}
        </div>
      </div>

      <div className="flex gap-3 items-center text-right">
        {stage.status === "running" && (
          <Badge variant="warning">
            {t("stage_running")}
          </Badge>
        )}
        {stage.status === "succeeded" && stage.duration && (
          <span className="font-mono text-muted-foreground text-sm">
            {stage.duration}
          </span>
        )}
        {stage.status === "skipped" && (
          <Badge variant="outline" className={getScanStatusClasses(stage.status)}>
            {stage.skipReason === "user_disabled" ? t("stage_skipped_user_disabled") :
              stage.skipReason === "target_not_applicable" ? t("stage_skipped_target_not_applicable") :
                t("stage_skipped")}
          </Badge>
        )}
        {stage.status === "pending" && (
          <span className="text-muted-foreground text-sm">{t("stage_pending")}</span>
        )}
        {stage.status === "failed" && (
          <Badge variant="error">
            {t("stage_failed")}
          </Badge>
        )}
        {stage.status === "cancelled" && (
          <Badge variant="warning">
            {t("stage_cancelled")}
          </Badge>
        )}
      </div>
    </div>
  )
}

interface ScanProgressSummaryProps {
  data: ScanProgressData
  engineNames: string[]
  locale: string
  t: TranslationFn
}

export function ScanProgressSummary({ data, engineNames, locale, t }: ScanProgressSummaryProps) {
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">{t("target")}</span>
        <span className="font-medium">{data.target?.name}</span>
      </div>
      <div className="flex gap-4 items-start justify-between text-sm">
        <span className="shrink-0 text-muted-foreground">{t("workflow")}</span>
        <div className="flex flex-wrap gap-1.5 justify-end">
          {engineNames.length ? (
            engineNames.map((name) => (
              <Badge key={name} variant="secondary" className="text-xs whitespace-nowrap">
                {name}
              </Badge>
            ))
          ) : (
            <span className="text-muted-foreground">-</span>
          )}
        </div>
      </div>
      {data.startedAt && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">{t("startTime")}</span>
          <span className="font-mono text-xs">{formatDateTime(data.startedAt, locale)}</span>
        </div>
      )}
      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">{t("status")}</span>
        <ScanStatusBadge status={data.status} t={t} />
      </div>
      {data.errorMessage && (
        <div className="bg-destructive/10 border border-destructive/20 mt-2 p-3 rounded-md">
          <p className={cn(textRole.bodyStrong, "text-destructive")}>{t("errorReason")}</p>
          <p className={cn("break-words mt-1", textRole.body, "text-destructive/80")}>{data.errorMessage}</p>
        </div>
      )}
    </div>
  )
}

interface ScanProgressTabsProps {
  activeTab: "stages" | "logs"
  onChange: (tab: "stages" | "logs") => void
  t: TranslationFn
}

export function ScanProgressTabs({ activeTab, onChange, t }: ScanProgressTabsProps) {
  return (
    <Tabs value={activeTab} onValueChange={(value) => onChange(value as "stages" | "logs")}>
      <TabsList className="grid grid-cols-2 w-full">
        <TabsTrigger value="stages">{t("tab_stages")}</TabsTrigger>
        <TabsTrigger value="logs">{t("tab_logs")}</TabsTrigger>
      </TabsList>
    </Tabs>
  )
}

interface ScanProgressStageListProps {
  stages: StageDetail[]
  engineNamesById: ReadonlyMap<string, string>
  t: TranslationFn
}

export function ScanProgressStageList({ stages, engineNamesById, t }: ScanProgressStageListProps) {
  return (
    <div className="max-h-[300px] overflow-y-auto space-y-2">
      {stages.map((stage) => (
        <StageRow key={`${stage.stage}-${stage.engineId}`} stage={stage} engineName={engineNamesById.get(stage.engineId)!} t={t} />
      ))}
    </div>
  )
}

interface ScanProgressLogsPanelProps {
  logs: ScanLog[]
  loading: boolean
}

export function ScanProgressLogsPanel({ logs, loading }: ScanProgressLogsPanelProps) {
  return (
    <div className="h-[300px] overflow-hidden rounded-md">
      <ScanLogList logs={logs} loading={loading} />
    </div>
  )
}
