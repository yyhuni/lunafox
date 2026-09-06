import React from "react"
import Link from "next/link"
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
import { getSeverityColor } from "@/lib/severity-config"
import {
  getScanStatusBadgeVariant,
  getScanStatusClasses,
  getStatusToneBgClass,
} from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { serializeWorkflowConfiguration } from "@/lib/workflow-config"
import { cn } from "@/lib/utils"
import type { ScanRecord } from "@/types/scan.types"
import type { ScanOverviewState } from "@/components/scan/history/scan-overview-state"
import {
  formatDate,
  formatDuration,
  getStatusIcon,
} from "@/components/scan/history/scan-overview-utils"
import {
  RuntimeConfigurationPanel,
  RuntimeDetailTabsPanel,
  type RuntimeDetailTab,
} from "@/components/scan/history/scan-runtime-detail-drawer"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface ScanOverviewRuntimeDetailsProps {
  t: TranslationFn
  scan: ScanRecord
  detailTab: RuntimeDetailTab
  setDetailTab: (value: RuntimeDetailTab) => void
  logs: ScanOverviewState["logs"]
  logsLoading: boolean
  isRunning: boolean
  autoRefresh: boolean
  setAutoRefresh: (value: boolean) => void
}

export function ScanOverviewRuntimeDetails({
  t,
  scan,
  detailTab,
  setDetailTab,
  logs,
  logsLoading,
  isRunning,
  autoRefresh,
  setAutoRefresh,
}: ScanOverviewRuntimeDetailsProps) {
  const yamlContent = serializeWorkflowConfiguration(scan.configuration)
  const hasRuntimeLogs = Boolean(scan.runtimeTasks?.some((task) => task.status !== "skipped" || task.startedAt || task.completedAt))
  const skipReason = scan.runtimeTasks?.find((task) => task.status === "skipped")?.skipReason

  return (
    <RuntimeDetailTabsPanel
      t={t}
      detailTab={detailTab}
      setDetailTab={setDetailTab}
      logs={logs}
      logsLoading={logsLoading}
      hasRuntimeLogs={hasRuntimeLogs}
      skipReason={skipReason}
      yamlContent={yamlContent}
      headerTrailing={
        <>
          {detailTab === "logs" && isRunning ? (
            <div className="flex flex-wrap items-center gap-3 self-start sm:self-auto">
              <div className="flex items-center gap-2">
                <Switch
                  id="overview-log-auto-refresh"
                  checked={autoRefresh}
                  onCheckedChange={setAutoRefresh}
                  className="scale-75"
                />
                <Label htmlFor="overview-log-auto-refresh" className="cursor-pointer text-xs">
                  {t("autoRefresh")}
                </Label>
              </div>
            </div>
          ) : null}
        </>
      }
    />
  )
}

interface ScanOverviewHeaderProps {
  t: TranslationFn
  tStatus: TranslationFn
  locale: string
  scan: ScanRecord
  executedEngineNames: string[]
  startedAt?: string
  completedAt?: string
}

export function ScanOverviewHeader({
  t,
  tStatus,
  locale,
  scan,
  executedEngineNames,
  startedAt,
  completedAt,
}: ScanOverviewHeaderProps) {
  const statusIconConfig = React.useMemo(() => getStatusIcon(scan.status ?? "pending"), [scan.status])
  const StatusIcon = statusIconConfig.icon
  const statusStyle = getScanStatusClasses(scan.status ?? "pending")
  const statusVariant = getScanStatusBadgeVariant(scan.status ?? "pending")

  return (
    <div className="flex items-center justify-between">
      <div className={cn("flex gap-6 items-center", textRole.bodySubtle)}>
        <div className="flex gap-1.5 items-center">
          <Calendar className="h-4 w-4" />
          <span>{t("startedAt")}: {formatDate(startedAt, locale)}</span>
        </div>
        <div className="flex gap-1.5 items-center">
          <Clock className="h-4 w-4" />
          <span>{t("duration")}: {formatDuration(startedAt, completedAt)}</span>
        </div>
        {executedEngineNames.length > 0 && (
          <div className="flex gap-1.5 items-center">
            <Cpu className="h-4 w-4" />
            <span>{executedEngineNames.join(", ")}</span>
          </div>
        )}
        {scan.agentName && (
          <div className="flex gap-1.5 items-center">
            <HardDrive className="h-4 w-4" />
            <span>{scan.agentName}</span>
          </div>
        )}
        <div className="flex gap-1.5 items-center">
          <span>{t("inputSource.label")}:</span>
          <span>{t(`inputSource.${scan.inputSource}`)}</span>
        </div>
      </div>
      <Badge variant={statusVariant} className={statusVariant === "outline" ? statusStyle : undefined}>
        <StatusIcon className={`h-3.5 w-3.5 mr-1.5 ${statusIconConfig.animate ? "animate-spin" : ""}`} />
        {tStatus(scan.status)}
      </Badge>
    </div>
  )
}

interface ScanOverviewAssetsProps {
  t: TranslationFn
  assetCards: ScanOverviewState["assetCards"]
  showTitle?: boolean
}

export function ScanOverviewAssets({ t, assetCards, showTitle = true }: ScanOverviewAssetsProps) {
  return (
    <div>
      {showTitle ? <h3 className={cn("mb-4", textRole.sectionTitle)}>{t("assetsTitle")}</h3> : null}
      <div className="gap-4 grid lg:grid-cols-6 md:grid-cols-2">
        {assetCards.map((card) => (
          <Link key={card.title} href={card.href} className="block">
            <div
              className="bg-card cursor-pointer duration-300 group hover:bg-accent/5 p-4 relative transition-[background-color,border-color,box-shadow]"
            >
              <div className="absolute border border-border/40 group-hover:border-primary/30 inset-0 transition-colors" />
              <div className="absolute border-primary/50 border-r border-t h-2 right-0 top-0 w-2" />
              <div className="absolute border-b border-l border-primary/50 bottom-0 h-2 left-0 w-2" />

              <div className="relative z-10">
                <div className="flex items-start justify-between mb-2">
                  <div className="bg-muted font-mono px-1.5 py-0.5 rounded-sm text-[10px] text-muted-foreground">
                    {card.code}
                  </div>
                  <card.icon className="group-hover:text-primary h-4 text-muted-foreground/70 transition-colors w-4" />
                </div>

                <div className={cn("duration-300 group-hover:text-primary transition-colors", textRole.metricValueDisplay)}>
                  {card.value.toLocaleString()}
                </div>

                <div className="flex gap-2 items-center mt-2">
                  <div className="bg-border border-dashed border-muted-foreground/20 border-t flex-1 h-px" />
                  <span className="font-mono text-[11px] text-foreground/85 tracking-wider uppercase">
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

interface ScanVulnerabilitySummaryProps {
  t: TranslationFn
  scanId: number
  vulnSummary: ScanOverviewState["vulnSummary"]
}

export function ScanVulnerabilitySummary({ t, scanId, vulnSummary }: ScanVulnerabilitySummaryProps) {
  return (
    <Link href={`/scan/history/${scanId}/vulnerabilities/`} className="block">
      <Card className="cursor-pointer hover:border-primary/50 transition-colors">
        <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
          <CardTitle>{t("vulnerabilitiesTitle")}</CardTitle>
          <ChevronRight className="h-4 text-muted-foreground w-4" />
        </CardHeader>
        <CardContent className="pt-0">
          <div className="flex flex-wrap gap-3 items-center">
            <div className="flex gap-1.5 items-center">
              <span aria-hidden="true" className="h-2.5 rounded-full w-2.5" style={{ backgroundColor: getSeverityColor("critical") }} />
              <span className={textRole.metadataValueStrong}>{vulnSummary.critical}</span>
            </div>
            <div className="flex gap-1.5 items-center">
              <span aria-hidden="true" className="h-2.5 rounded-full w-2.5" style={{ backgroundColor: getSeverityColor("high") }} />
              <span className={textRole.metadataValueStrong}>{vulnSummary.high}</span>
            </div>
            <div className="flex gap-1.5 items-center">
              <span aria-hidden="true" className="h-2.5 rounded-full w-2.5" style={{ backgroundColor: getSeverityColor("medium") }} />
              <span className={textRole.metadataValueStrong}>{vulnSummary.medium}</span>
            </div>
            <div className="flex gap-1.5 items-center">
              <span aria-hidden="true" className="h-2.5 rounded-full w-2.5" style={{ backgroundColor: getSeverityColor("low") }} />
              <span className={textRole.metadataValueStrong}>{vulnSummary.low}</span>
            </div>
            <span className={cn("ml-auto", textRole.helperText)}>
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
  configuration?: ScanRecord["configuration"] | null
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
  configuration,
}: ScanLogsPanelProps) {
  const yamlContent = serializeWorkflowConfiguration(configuration)
  return (
    <div className="border flex flex-col min-h-0 overflow-hidden rounded-lg">
      <div className="bg-muted/30 border-b flex items-center justify-between px-3 py-2 shrink-0">
        <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as "logs" | "config")}>
          <TabsList variant="content">
            <TabsTrigger variant="content" value="logs">
              {t("logsTitle")}
            </TabsTrigger>
            <TabsTrigger variant="content" value="config">
              {t("configTitle")}
            </TabsTrigger>
          </TabsList>
        </Tabs>
        {activeTab === "logs" && isRunning && (
          <div className="flex gap-2 items-center">
            <Switch
              id="log-auto-refresh"
              checked={autoRefresh}
              onCheckedChange={setAutoRefresh}
              className="scale-75"
            />
            <Label htmlFor="log-auto-refresh" className="cursor-pointer text-xs">
              {t("autoRefresh")}
            </Label>
          </div>
        )}
      </div>

      <div className="flex-1 min-h-0">
        {activeTab === "logs" ? (
          <ScanLogList logs={logs} loading={logsLoading} />
        ) : (
          <RuntimeConfigurationPanel yamlContent={yamlContent} t={t} />
        )}
      </div>

      {activeTab === "logs" && (
        <div className="bg-muted/50 border-t flex items-center px-4 py-2 shrink-0 text-muted-foreground text-xs">
          <span>{t("logCount", { count: logs.length })}</span>
          {isRunning && autoRefresh && (
            <>
              <Separator orientation="vertical" className="h-3 mx-3" />
              <span className="flex gap-1.5 items-center">
                <span className={cn("animate-pulse rounded-full size-1.5", getStatusToneBgClass("success"))} />
                {t("autoRefreshEvery")}
              </span>
            </>
          )}
        </div>
      )}
    </div>
  )
}
