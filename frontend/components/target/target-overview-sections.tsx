"use client"

import React from "react"
import Link from "next/link"
import dynamic from "next/dynamic"
import {
  Globe,
  Network,
  Server,
  Link2,
  FolderOpen,
  Camera,
  ShieldAlert,
  AlertTriangle,
  Clock,
  Calendar,
  ChevronRight,
  CheckCircle2,
  PauseCircle,
  Loader2,
} from "@/components/icons"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Button } from "@/components/ui/button"
import { ScanHistoryList } from "@/components/scan/history/scan-history-list"

import type { TargetOverviewState } from "./target-overview-state"

const InitiateScanDialog = dynamic(
  () => import("@/components/scan/initiate-scan-dialog").then((m) => ({ default: m.InitiateScanDialog })),
  { ssr: false }
)

const ASSET_ICON_MAP = {
  websites: Globe,
  subdomains: Network,
  ips: Server,
  urls: Link2,
  directories: FolderOpen,
  screenshots: Camera,
} as const

type AssetIconKey = keyof typeof ASSET_ICON_MAP

export function TargetOverviewLoadingState() {
  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {[...Array(6)].map((_, i) => (
          <Card key={i}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-4 w-4" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-16" />
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}

export function TargetOverviewErrorState({
  t,
}: {
  t: (key: string) => string
}) {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <AlertTriangle className="h-10 w-10 text-destructive mb-4" />
      <p className="text-muted-foreground">{t("loadError")}</p>
    </div>
  )
}

export function TargetOverviewContent({
  state,
}: {
  state: TargetOverviewState
}) {
  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-3 -mt-4">
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-muted-foreground">
          <div className="flex items-center gap-1.5">
            <Calendar className="h-4 w-4" />
            <span>{state.t("createdAt")}: {state.formatDate(state.target?.createdAt)}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <Clock className="h-4 w-4" />
            <span>{state.t("lastScanned")}: {state.formatDate(state.target?.lastScannedAt)}</span>
          </div>
        </div>
        <Button
          onClick={() => state.setScanDialogOpen(true)}
          className="
            relative overflow-hidden group
            bg-background border border-destructive/50 text-destructive
            hover:border-destructive
            min-w-[140px]
            shrink-0 self-end
          "
        >
          <span className="absolute inset-y-0 left-0 w-[2px] bg-destructive transition-[width,opacity] duration-300 group-hover:w-full opacity-10" />
          <span className="absolute bottom-0 right-0 h-[2px] w-0 bg-destructive transition-[width] duration-300 group-hover:w-full" />
          <span className="relative z-10 flex items-center gap-2">
            <Loader2 className="h-4 w-4 animate-spin hidden group-hover:block" />
            <span className="group-hover:hidden">{state.t("initiateScan")}</span>
            <span className="hidden group-hover:block">{state.tInitiate("initiating")}</span>
          </span>
        </Button>
      </div>

      <div>
        <h3 className="text-lg font-semibold mb-4">{state.t("assetsTitle")}</h3>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-6">
          {state.assetCards.map((card) => {
            const Icon = ASSET_ICON_MAP[card.iconKey as AssetIconKey]
            return (
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
                      <Icon className="h-4 w-4 text-muted-foreground/70 group-hover:text-primary transition-colors" />
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
            )
          })}
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card className="flex flex-col">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-3">
            <div className="flex items-center gap-2">
              <Clock className="h-4 w-4 text-muted-foreground" />
              <CardTitle className="text-sm font-medium">{state.t("scheduledScans.title")}</CardTitle>
            </div>
            <Link href={`/target/${state.targetId}/settings/`}>
              <Button variant="ghost" size="sm" className="h-7 text-xs">
                {state.t("scheduledScans.manage")}
                <ChevronRight className="h-3 w-3 ml-1" />
              </Button>
            </Link>
          </CardHeader>
          <CardContent className="flex-1 flex flex-col">
            {state.isLoadingScans ? (
              <div className="space-y-2">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-4 w-48" />
              </div>
            ) : state.totalScheduledScans === 0 ? (
              <div className="flex-1 flex flex-col items-center justify-center">
                <Clock className="h-8 w-8 text-muted-foreground/50 mb-2" />
                <p className="text-sm text-muted-foreground">{state.t("scheduledScans.empty")}</p>
                <Link href={`/target/${state.targetId}/settings/`}>
                  <Button variant="link" size="sm" className="mt-1">
                    {state.t("scheduledScans.createFirst")}
                  </Button>
                </Link>
              </div>
            ) : (
              <div className="space-y-3">
                <div className="flex items-center gap-4 text-sm">
                  <div>
                    <span className="text-muted-foreground">{state.t("scheduledScans.configured")}: </span>
                    <span className="font-medium">{state.totalScheduledScans}</span>
                  </div>
                  <div>
                    <span className="text-muted-foreground">{state.t("scheduledScans.enabled")}: </span>
                    <span className="font-medium text-green-600">{state.enabledScans.length}</span>
                  </div>
                </div>

                {state.nextExecution ? (
                  <div className="text-sm">
                    <span className="text-muted-foreground">{state.t("scheduledScans.nextRun")}: </span>
                    <span className="font-medium">
                      {state.formatShortDate(
                        state.nextExecution.nextRunTime,
                        state.t("scheduledScans.today"),
                        state.t("scheduledScans.tomorrow")
                      )}
                    </span>
                  </div>
                ) : null}

                <div className="space-y-2 pt-2 border-t">
                  {state.scheduledScans.slice(0, 2).map((scan) => (
                    <div key={scan.id} className="flex items-center gap-2 text-sm">
                      {scan.isEnabled ? (
                        <CheckCircle2 className="h-3.5 w-3.5 text-green-500 shrink-0" />
                      ) : (
                        <PauseCircle className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                      )}
                      <span className={`truncate ${!scan.isEnabled ? "text-muted-foreground" : ""}`}>
                        {scan.name}
                      </span>
                    </div>
                  ))}
                  {state.totalScheduledScans > 2 ? (
                    <p className="text-xs text-muted-foreground">
                      {state.t("scheduledScans.more", { count: state.totalScheduledScans - 2 })}
                    </p>
                  ) : null}
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        <Link href={`/target/${state.targetId}/vulnerabilities/`} className="block">
          <Card className="h-full hover:border-primary/50 transition-colors cursor-pointer flex flex-col">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-3">
              <div className="flex items-center gap-2">
                <ShieldAlert className="h-4 w-4 text-muted-foreground" />
                <CardTitle className="text-sm font-medium">{state.t("vulnerabilitiesTitle")}</CardTitle>
              </div>
              <Button variant="ghost" size="sm" className="h-7 text-xs">
                {state.t("viewAll")}
                <ChevronRight className="h-3 w-3 ml-1" />
              </Button>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-baseline gap-2">
                <span className="text-3xl font-bold">{state.vulnSummary.total}</span>
                <span className="text-sm text-muted-foreground">{state.t("cards.vulnerabilities")}</span>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="flex items-center gap-2">
                  <div className="w-3 h-3 rounded-full bg-red-500" />
                  <span className="text-sm text-muted-foreground">{state.t("severity.critical")}</span>
                  <span className="text-sm font-medium ml-auto">{state.vulnSummary.critical}</span>
                </div>
                <div className="flex items-center gap-2">
                  <div className="w-3 h-3 rounded-full bg-orange-500" />
                  <span className="text-sm text-muted-foreground">{state.t("severity.high")}</span>
                  <span className="text-sm font-medium ml-auto">{state.vulnSummary.high}</span>
                </div>
                <div className="flex items-center gap-2">
                  <div className="w-3 h-3 rounded-full bg-yellow-500" />
                  <span className="text-sm text-muted-foreground">{state.t("severity.medium")}</span>
                  <span className="text-sm font-medium ml-auto">{state.vulnSummary.medium}</span>
                </div>
                <div className="flex items-center gap-2">
                  <div className="w-3 h-3 rounded-full bg-blue-500" />
                  <span className="text-sm text-muted-foreground">{state.t("severity.low")}</span>
                  <span className="text-sm font-medium ml-auto">{state.vulnSummary.low}</span>
                </div>
              </div>
            </CardContent>
          </Card>
        </Link>
      </div>

      <div>
        <h3 className="text-lg font-semibold mb-4">{state.t("scanHistoryTitle")}</h3>
        <ScanHistoryList
          targetId={state.targetId}
          hideToolbar
          pageSize={5}
          hideTargetColumn
          pageSizeOptions={[5, 10, 20, 50, 100]}
        />
      </div>
    </div>
  )
}

export function TargetOverviewDialog({
  state,
}: {
  state: TargetOverviewState
}) {
  return (
    <InitiateScanDialog
      open={state.scanDialogOpen}
      onOpenChange={state.setScanDialogOpen}
      targetId={state.targetId}
      targetName={state.target?.name ?? ""}
    />
  )
}
