"use client"

import type { ScanProgressData, StageDetail } from "@/components/scan/scan-progress-dialog-types"
import type { StageStatus } from "@/types/scan.types"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  IconCircleCheck,
  IconCircleX,
  IconClock,
} from "@/components/icons"
import { cn } from "@/lib/utils"
import { ScanLogList } from "@/components/scan/scan-log-list"
import type { ScanLog } from "@/services/scan.service"
import { formatDateTime } from "@/components/scan/scan-progress-dialog-utils"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

const SCAN_STATUS_STYLES: Record<string, string> = {
  running: "bg-[var(--warning)]/10 text-[var(--warning)] border-[var(--warning)]/20",
  cancelled: "bg-[var(--muted-foreground)]/10 text-[var(--muted-foreground)] border-[var(--muted-foreground)]/20",
  completed: "bg-[var(--success)]/10 text-[var(--success)] border-[var(--success)]/20",
  failed: "bg-[var(--error)]/10 text-[var(--error)] border-[var(--error)]/20",
  initiated: "bg-[var(--warning)]/10 text-[var(--warning)] border-[var(--warning)]/20",
}

function PulsingDot({ className }: { className?: string }) {
  return (
    <span className={cn("relative flex h-3 w-3", className)}>
      <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-current opacity-75" />
      <span className="relative inline-flex h-3 w-3 rounded-full bg-current" />
    </span>
  )
}

export function ScanStatusIcon({ status }: { status: string }) {
  switch (status) {
    case "running":
      return <PulsingDot className="text-[var(--warning)]" />
    case "completed":
      return <IconCircleCheck className="h-5 w-5 text-[var(--success)]" />
    case "cancelled":
      return <IconCircleX className="h-5 w-5 text-[var(--muted-foreground)]" />
    case "failed":
      return <IconCircleX className="h-5 w-5 text-[var(--error)]" />
    case "pending":
      return <PulsingDot className="text-[var(--warning)]" />
    default:
      return <PulsingDot className="text-muted-foreground" />
  }
}

export function ScanStatusBadge({ status, t }: { status: string; t: TranslationFn }) {
  const className = SCAN_STATUS_STYLES[status] || "bg-muted text-muted-foreground"
  const label = t(`status_${status}`)
  return (
    <Badge variant="outline" className={className}>
      {label}
    </Badge>
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
      return <IconCircleX className="h-5 w-5 text-[var(--warning)]" />
    default:
      return <IconClock className="h-5 w-5 text-muted-foreground" />
  }
}

function StageRow({ stage, t }: { stage: StageDetail; t: TranslationFn }) {
  return (
    <div
      className={cn(
        "flex items-center justify-between py-3 px-4 rounded-lg transition-colors",
        stage.status === "running" && "bg-[var(--warning)]/10 border border-[var(--warning)]/20",
        stage.status === "completed" && "bg-muted/50",
        stage.status === "failed" && "bg-[var(--error)]/10",
        stage.status === "cancelled" && "bg-[var(--warning)]/10",
      )}
    >
      <div className="flex items-center gap-3">
        <StageStatusIcon status={stage.status} />
        <div>
          <span className="font-medium">{t(`stages.${stage.stage}`)}</span>
          {stage.detail && (
            <p className="text-xs text-muted-foreground mt-0.5">
              {stage.detail}
            </p>
          )}
        </div>
      </div>

      <div className="flex items-center gap-3 text-right">
        {stage.status === "running" && (
          <Badge variant="outline" className="bg-[var(--warning)]/10 text-[var(--warning)] border-[var(--warning)]/20">
            {t("stage_running")}
          </Badge>
        )}
        {stage.status === "completed" && stage.duration && (
          <span className="text-sm text-muted-foreground font-mono">
            {stage.duration}
          </span>
        )}
        {stage.status === "pending" && (
          <span className="text-sm text-muted-foreground">{t("stage_pending")}</span>
        )}
        {stage.status === "failed" && (
          <Badge variant="outline" className="bg-[var(--error)]/10 text-[var(--error)] border-[var(--error)]/20">
            {t("stage_failed")}
          </Badge>
        )}
        {stage.status === "cancelled" && (
          <Badge variant="outline" className="bg-[var(--warning)]/10 text-[var(--warning)] border-[var(--warning)]/20">
            {t("stage_cancelled")}
          </Badge>
        )}
      </div>
    </div>
  )
}

interface ScanProgressSummaryProps {
  data: ScanProgressData
  locale: string
  t: TranslationFn
}

export function ScanProgressSummary({ data, locale, t }: ScanProgressSummaryProps) {
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">{t("target")}</span>
        <span className="font-medium">{data.target?.name}</span>
      </div>
      <div className="flex items-start justify-between text-sm gap-4">
        <span className="text-muted-foreground shrink-0">{t("engine")}</span>
        <div className="flex flex-wrap gap-1.5 justify-end">
          {data.engineNames?.length ? (
            data.engineNames.map((name) => (
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
        <div className="mt-2 p-3 bg-destructive/10 border border-destructive/20 rounded-md">
          <p className="text-sm text-destructive font-medium">{t("errorReason")}</p>
          <p className="text-sm text-destructive/80 mt-1 break-words">{data.errorMessage}</p>
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
      <TabsList className="grid w-full grid-cols-2">
        <TabsTrigger value="stages">{t("tab_stages")}</TabsTrigger>
        <TabsTrigger value="logs">{t("tab_logs")}</TabsTrigger>
      </TabsList>
    </Tabs>
  )
}

interface ScanProgressStageListProps {
  stages: StageDetail[]
  t: TranslationFn
}

export function ScanProgressStageList({ stages, t }: ScanProgressStageListProps) {
  return (
    <div className="space-y-2 max-h-[300px] overflow-y-auto">
      {stages.map((stage) => (
        <StageRow key={stage.stage} stage={stage} t={t} />
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
