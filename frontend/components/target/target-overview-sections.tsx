"use client"

import React from "react"
import Link from "next/link"
import dynamic from "next/dynamic"
import {
  Clock,
  Calendar,
  ChevronRight,
  CheckCircle2,
  PauseCircle,
  semanticIcons,
} from "@/components/icons"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Button } from "@/components/ui/button"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { ScanHistoryList } from "@/components/scan/history/scan-history-list"
import { ScanHistoryListLoadingState } from "@/components/scan/history/scan-history-list-sections"
import { getSeverityColor } from "@/lib/severity-config"
import { getStatusToneTextClass } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import {
  deferredInteractionUnmountDelayMs,
  useDeferredInteractionMount,
} from "@/hooks/use-deferred-interaction-mount"

import type { TargetOverviewState } from "./target-overview-state"

const VulnerabilityIcon = semanticIcons.concept.vulnerability
const ScanIcon = semanticIcons.concept.scan

const InitiateScanDrawer = dynamic(
  () => import("@/components/scan/initiate-scan-dialog").then((m) => ({ default: m.InitiateScanDrawer })),
  { ssr: false, loading: () => null }
)

const ASSET_ICON_MAP = {
  websites: semanticIcons.concept.website,
  subdomains: semanticIcons.concept.subdomain,
  ips: semanticIcons.concept.ip,
  urls: semanticIcons.concept.endpoint,
  directories: semanticIcons.concept.directory,
  screenshots: semanticIcons.concept.screenshot,
} as const

type AssetIconKey = keyof typeof ASSET_ICON_MAP
type SummarySeverity = "critical" | "high" | "medium" | "low"

function TargetOverviewTopBar({ children }: { children: React.ReactNode }) {
  return (
    <div className="-mt-4 flex flex-wrap gap-3 items-end justify-between">
      {children}
    </div>
  )
}

function TargetOverviewTextPlaceholder({
  roleClassName,
  className,
  featured = false,
}: {
  roleClassName: string
  className?: string
  featured?: boolean
}) {
  return (
    <span
      aria-hidden="true"
      data-featured={featured ? "true" : undefined}
      className={cn("relative block", roleClassName, className)}
    >
      <span className="invisible">{"\u00a0"}</span>
      <Skeleton className="absolute inset-0" />
    </span>
  )
}

function TargetOverviewSection({
  title,
  children,
}: {
  title: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <div>
      <h3 className={cn("mb-4", textRole.sectionTitle)}>{title}</h3>
      {children}
    </div>
  )
}

function TargetAssetCardGrid({ children }: { children: React.ReactNode }) {
  return (
    <div className="gap-4 grid lg:grid-cols-6 md:grid-cols-2">
      {children}
    </div>
  )
}

function TargetAssetCardShell({
  interactive = false,
  children,
}: {
  interactive?: boolean
  children: React.ReactNode
}) {
  return (
    <div
      className={cn(
        "bg-card p-4 relative",
        interactive && "cursor-pointer duration-300 group hover:bg-accent/5 transition-[background-color,border-color,box-shadow]"
      )}
    >
      <div className={cn("absolute border border-border/40 inset-0", interactive && "group-hover:border-primary/30 transition-colors")} />
      <div className="absolute border-primary/50 border-r border-t h-2 right-0 top-0 w-2" />
      <div className="absolute border-b border-l border-primary/50 bottom-0 h-2 left-0 w-2" />
      <div className="relative z-10">
        {children}
      </div>
    </div>
  )
}

function TargetSummaryCardGrid({ children }: { children: React.ReactNode }) {
  return (
    <div className="gap-4 grid md:grid-cols-2">
      {children}
    </div>
  )
}

export function TargetOverviewLoadingState() {
  return (
    <div className="space-y-6">
      <TargetOverviewTopBar>
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
          <Skeleton className="h-5 w-40 rounded-full" />
          <Skeleton className="h-5 w-44 rounded-full" />
        </div>
        <ActionSkeleton widthClassName="w-36" />
      </TargetOverviewTopBar>

      <TargetOverviewSection title={<TargetOverviewTextPlaceholder roleClassName={textRole.sectionTitle} className="w-24" />}>
        <TargetAssetCardGrid>
          {[...Array(6)].map((_, i) => (
            <TargetAssetCardShell key={i}>
              <div className="space-y-3">
                <div className="flex items-start justify-between">
                  <TargetOverviewTextPlaceholder roleClassName={textRole.code} className="w-12" />
                  <Skeleton className="size-4 rounded-full" />
                </div>
                <TargetOverviewTextPlaceholder roleClassName={textRole.metricValueDisplay} className="w-16" />
                <div className="flex gap-2 items-center">
                  <div className="bg-border border-dashed border-muted-foreground/20 border-t flex-1 h-px" />
                  <TargetOverviewTextPlaceholder roleClassName={textRole.metadataValue} className="w-16" />
                </div>
              </div>
            </TargetAssetCardShell>
          ))}
        </TargetAssetCardGrid>
      </TargetOverviewSection>

      <TargetSummaryCardGrid>
        {[...Array(2)].map((_, i) => (
          <Card key={i}>
            <CardHeader className="flex flex-row items-center justify-between pb-3 space-y-0">
              <TargetOverviewTextPlaceholder roleClassName={textRole.sectionTitle} className="w-24" />
              <Skeleton className="h-4 w-4" />
            </CardHeader>
            <CardContent>
              {i === 1 ? (
                <TargetOverviewTextPlaceholder featured roleClassName={textRole.metricValueDisplay} className="w-16" />
              ) : (
                <div className="space-y-2">
                  <TargetOverviewTextPlaceholder roleClassName={textRole.metadataValue} className="w-16" />
                  <TargetOverviewTextPlaceholder roleClassName={textRole.metadataValue} className="w-24" />
                </div>
              )}
            </CardContent>
          </Card>
        ))}
      </TargetSummaryCardGrid>

      <TargetOverviewSection title={<TargetOverviewTextPlaceholder roleClassName={textRole.sectionTitle} className="w-28" />}>
        <ScanHistoryListLoadingState
          owner="target-overview-scan-history"
          hideToolbar
          rowCount={5}
          hideTargetColumn
          layer="section"
        />
      </TargetOverviewSection>
    </div>
  )
}

export function TargetOverviewContent({
  state,
}: {
  state: TargetOverviewState
}) {
  const severityRows: Array<{ level: SummarySeverity; label: string; count: number }> = [
    { level: "critical", label: state.t("severity.critical"), count: state.vulnSummary.critical },
    { level: "high", label: state.t("severity.high"), count: state.vulnSummary.high },
    { level: "medium", label: state.t("severity.medium"), count: state.vulnSummary.medium },
    { level: "low", label: state.t("severity.low"), count: state.vulnSummary.low },
  ]

  return (
    <div className="space-y-6">
      <TargetOverviewTopBar>
        <div className={cn("flex flex-wrap gap-x-4 gap-y-2 items-center", textRole.bodySubtle)}>
          <div className="flex gap-1.5 items-center">
            <Calendar className="h-4 w-4" />
            <span>{state.t("createdAt")}: {state.formatDate(state.target?.createdAt)}</span>
          </div>
          <div className="flex gap-1.5 items-center">
            <Clock className="h-4 w-4" />
            <span>{state.t("lastScanned")}: {state.formatDate(state.target?.lastScannedAt)}</span>
          </div>
        </div>
        <Button
          onClick={() => state.setScanDialogOpen(true)}
          className="bg-background border border-destructive/50 group hover:bg-destructive/10 hover:border-destructive hover:text-destructive active:bg-destructive/15 min-w-[140px] overflow-hidden relative self-end shrink-0 text-destructive"
        >
          <span className="absolute bg-destructive duration-300 group-hover:opacity-20 inset-y-0 left-0 opacity-10 transition-opacity w-[2px]" />
          <span className="absolute bg-destructive bottom-0 duration-300 group-hover:opacity-100 h-0.5 opacity-0 right-0 transition-opacity w-full" />
          <span className="flex gap-2 items-center relative z-10">
            <ScanIcon className="group-hover:block h-4 hidden w-4" />
            <span className="group-hover:hidden">{state.t("initiateScan")}</span>
            <span className="group-hover:block hidden">{state.tInitiate("initiating")}</span>
          </span>
        </Button>
      </TargetOverviewTopBar>

      <TargetOverviewSection title={state.t("assetsTitle")}>
        <TargetAssetCardGrid>
          {state.assetCards.map((card) => {
            const Icon = ASSET_ICON_MAP[card.iconKey as AssetIconKey]
            return (
              <Link key={card.title} href={card.href} className="block">
                <TargetAssetCardShell interactive>
                  <div className="flex items-start justify-between mb-2">
                    <div className={cn("bg-muted px-1.5 py-0.5 rounded-sm", textRole.code, "text-muted-foreground")}>
                      {card.code}
                    </div>
                    <Icon className="group-hover:text-primary h-4 text-muted-foreground/70 transition-colors w-4" />
                  </div>

                  <div className={cn("duration-300 group-hover:text-primary transition-colors", textRole.metricValueDisplay)}>
                    {card.value.toLocaleString()}
                  </div>

                  <div className="flex gap-2 items-center mt-2">
                    <div className="bg-border border-dashed border-muted-foreground/20 border-t flex-1 h-px" />
                    <span className={textRole.metadataValue}>
                      {card.title}
                    </span>
                  </div>
                </TargetAssetCardShell>
              </Link>
            )
          })}
        </TargetAssetCardGrid>
      </TargetOverviewSection>

      <TargetSummaryCardGrid>
        <Card className="flex flex-col">
          <CardHeader className="flex flex-row items-center justify-between pb-3 space-y-0">
            <div className="flex gap-2 items-center">
              <Clock className="h-4 text-muted-foreground w-4" />
              <CardTitle>{state.t("scheduledScans.title")}</CardTitle>
            </div>
            <Link href={`/targets/${state.targetId}/settings/scheduled-scans/`}>
              <Button variant="ghost" size="sm">
                {state.t("scheduledScans.manage")}
                <ChevronRight className="h-3 ml-1 w-3" />
              </Button>
            </Link>
          </CardHeader>
          <CardContent className="flex flex-1 flex-col">
            {state.isLoadingScans ? (
              <div className="space-y-2">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-4 w-48" />
              </div>
            ) : state.totalScheduledScans === 0 ? (
              <div className="flex flex-1 flex-col items-center justify-center">
                <Clock className="h-8 mb-2 text-muted-foreground/50 w-8" />
                <p className={textRole.bodySubtle}>{state.t("scheduledScans.empty")}</p>
                <Link href={`/targets/${state.targetId}/settings/scheduled-scans/`}>
                  <Button variant="link" size="sm" className="mt-1">
                    {state.t("scheduledScans.createFirst")}
                  </Button>
                </Link>
              </div>
            ) : (
              <div className="space-y-3">
                <div className="flex gap-4 items-center text-sm">
                  <div>
                    <span className={textRole.metadataLabel}>{state.t("scheduledScans.configured")}: </span>
                    <span className={textRole.metadataValue}>{state.totalScheduledScans}</span>
                  </div>
                  <div>
                    <span className={textRole.metadataLabel}>{state.t("scheduledScans.enabled")}: </span>
                    <span className={cn(textRole.metadataValue, getStatusToneTextClass("success"))}>{state.enabledScans.length}</span>
                  </div>
                </div>

                {state.nextExecution ? (
                  <div className="text-sm">
                    <span className={textRole.metadataLabel}>{state.t("scheduledScans.nextRun")}: </span>
                    <span className={textRole.metadataValue}>
                      {state.formatShortDate(
                        state.nextExecution.nextRunTime ?? undefined,
                        state.t("scheduledScans.today"),
                        state.t("scheduledScans.tomorrow")
                      )}
                    </span>
                  </div>
                ) : null}

                <div className="border-t pt-2 space-y-2">
                  {state.scheduledScans.slice(0, 2).map((scan) => (
                    <div key={scan.id} className="flex gap-2 items-center text-sm">
                      {scan.isEnabled ? (
                        <CheckCircle2 className={cn("h-3.5 w-3.5 shrink-0", getStatusToneTextClass("success"))} />
                      ) : (
                        <PauseCircle className="h-3.5 shrink-0 text-muted-foreground w-3.5" />
                      )}
                      <span className={`truncate ${!scan.isEnabled ? "text-muted-foreground" : ""}`}>
                        {scan.name}
                      </span>
                    </div>
                  ))}
                  {state.totalScheduledScans > 2 ? (
                    <p className={textRole.metadataLabel}>
                      {state.t("scheduledScans.more", { count: state.totalScheduledScans - 2 })}
                    </p>
                  ) : null}
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        <Link href={`/targets/${state.targetId}/vulnerabilities/`} className="block">
          <Card className="cursor-pointer flex flex-col h-full hover:border-primary/50 transition-colors">
            <CardHeader className="flex flex-row items-center justify-between pb-3 space-y-0">
              <div className="flex gap-2 items-center">
                <VulnerabilityIcon className="h-4 text-muted-foreground w-4" />
                <CardTitle>{state.t("vulnerabilitiesTitle")}</CardTitle>
              </div>
              <Button variant="ghost" size="sm">
                {state.t("viewAll")}
                <ChevronRight className="h-3 ml-1 w-3" />
              </Button>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex gap-2 items-baseline">
                <span data-featured="true" className={textRole.metricValueDisplay}>{state.vulnSummary.total}</span>
                <span className={textRole.bodySubtle}>{state.t("cards.vulnerabilities")}</span>
              </div>

              <div className="gap-3 grid grid-cols-2">
                {severityRows.map((row) => (
                  <div key={row.level} className="flex gap-2 items-center">
                    <span
                      aria-hidden="true"
                      className="h-3 rounded-full w-3"
                      style={{ backgroundColor: getSeverityColor(row.level) }}
                    />
                    <span className={textRole.bodySubtle}>{row.label}</span>
                    <span className={cn("ml-auto", textRole.metadataValue)}>{row.count}</span>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </Link>
      </TargetSummaryCardGrid>

      <TargetOverviewSection title={state.t("scanHistoryTitle")}>
        <ScanHistoryList
          targetId={state.targetId}
          hideToolbar
          pageSize={5}
          hideTargetColumn
          layer="section"
          pageSizeOptions={[5, 10, 20, 50, 100]}
        />
      </TargetOverviewSection>
    </div>
  )
}

export function TargetOverviewDialog({
  state,
}: {
  state: TargetOverviewState
}) {
  const shouldMountScanDrawer = useDeferredInteractionMount(state.scanDialogOpen, {
    unmountDelayMs: deferredInteractionUnmountDelayMs,
  })

  return shouldMountScanDrawer ? (
    <InitiateScanDrawer
      open={state.scanDialogOpen}
      onOpenChange={state.setScanDialogOpen}
      targetId={state.targetId}
      targetName={state.target?.name ?? ""}
    />
  ) : null
}
