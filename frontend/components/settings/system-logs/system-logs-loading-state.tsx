"use client"

import { useTranslations } from "next-intl"

import { PageHeader } from "@/components/common/page-header"
import {
  getLoadingOwnerAttributes,
  getLoadingStructureSlotAttributes,
} from "@/components/shared/loading/loading-owner"
import { TerminalLogToolbar } from "@/components/shared/visualization/terminal-log-toolbar"
import { LiveLogSurface } from "@/components/shared/visualization/terminal-log-surface"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { cn } from "@/lib/utils"
import {
  SYSTEM_LOGS_CONTENT_SHELL_CLASS,
  SYSTEM_LOGS_DEFAULT_LINES,
  SYSTEM_LOGS_FOOTER_OVERLAY_CLASS,
  SYSTEM_LOGS_FOOTER_OVERLAY_REFRESH_GROUP_CLASS,
  SYSTEM_LOGS_FOOTER_OVERLAY_STATUS_GROUP_CLASS,
  SYSTEM_LOGS_FOOTER_REFRESH_GROUP_CLASS,
  SYSTEM_LOGS_FOOTER_STATUS_GROUP_CLASS,
  SYSTEM_LOGS_HEADER_OVERLAY_SHELL_CLASS,
  SYSTEM_LOGS_HEADER_RULE_CLASS,
  SYSTEM_LOGS_HEADER_SHELL_CLASS,
  SYSTEM_LOGS_HEADER_TITLE_GROUP_CLASS,
  SYSTEM_LOGS_HEADER_TITLE_ROW_CLASS,
  SYSTEM_LOGS_PAGE_SHELL_CLASS,
  SYSTEM_LOGS_TERMINAL_SHELL_CLASS,
} from "./system-logs-layout"

export interface SystemLogsLoadingStateProps {
  pageTitle: string
  pageDescription: string
  owner?: string
  className?: string
}

export function SystemLogsLoadingState({
  pageTitle,
  pageDescription,
  owner,
  className,
}: SystemLogsLoadingStateProps) {
  const t = useTranslations("settings.systemLogs")

  if (owner !== undefined && !owner.trim()) {
    throw new Error("SystemLogsLoadingState requires a non-empty owner.")
  }

  return (
    <div
      {...(owner ? getLoadingOwnerAttributes({ owner, layer: "route", intent: "route" }) : {})}
      data-slot="system-logs-loading-state"
      className={cn(SYSTEM_LOGS_PAGE_SHELL_CLASS, className)}
    >
      <div
        {...getLoadingStructureSlotAttributes("system-logs-header")}
        className={SYSTEM_LOGS_HEADER_SHELL_CLASS}
      >
        <PageHeader
          code="LOG-01"
          title={pageTitle}
          description={pageDescription}
          className="opacity-0"
        />
        <div className={SYSTEM_LOGS_HEADER_OVERLAY_SHELL_CLASS} aria-hidden="true">
          <div className={SYSTEM_LOGS_HEADER_TITLE_ROW_CLASS}>
            <div className={SYSTEM_LOGS_HEADER_TITLE_GROUP_CLASS}>
              <Skeleton className="h-6 w-24" />
              <Skeleton className="h-4 w-16" />
            </div>
            <div className={SYSTEM_LOGS_HEADER_RULE_CLASS} />
          </div>
          <Skeleton className="h-4 w-80 max-w-full" />
        </div>
      </div>

      <div className={SYSTEM_LOGS_CONTENT_SHELL_CLASS}>
        <div className={SYSTEM_LOGS_TERMINAL_SHELL_CLASS}>
          <div {...getLoadingStructureSlotAttributes("system-logs-terminal-toolbar")} className="shrink-0">
            <TerminalLogToolbar
              searchTerm=""
              levelFilter="all"
              lineWindow={{
                value: SYSTEM_LOGS_DEFAULT_LINES,
                options: [100, 200, 500, 1000, 2000, 5000],
                onValueChange: () => {},
                label: t("toolbar.lineWindow"),
              }}
              levelLabels={{
                all: t("logs.filterAll"),
                error: t("logs.filterError"),
                warn: t("logs.filterWarn"),
                info: t("logs.filterInfo"),
                debug: t("logs.filterDebug"),
              }}
              loading
            />
          </div>

          <div {...getLoadingStructureSlotAttributes("system-logs-log-surface")} className="flex min-h-0 flex-1 flex-col">
            <LiveLogSurface
              footer={<>
              <div className={cn(SYSTEM_LOGS_FOOTER_STATUS_GROUP_CLASS, "opacity-0")}>
                <span className="shrink-0 whitespace-nowrap">{SYSTEM_LOGS_DEFAULT_LINES} {t("toolbar.linesUnit")}</span>
                <Separator orientation="vertical" className="hidden h-3 sm:block" />
                <span className="shrink-0 whitespace-nowrap">{t("toolbar.serverSource")}</span>
                <Separator orientation="vertical" className="hidden h-3 sm:block" />
                <span className="flex min-w-0 flex-1 items-center gap-1.5">
                  <span className="size-1.5 rounded-full bg-success" />
                  {t("description")}
                </span>
              </div>
              <div className={cn(SYSTEM_LOGS_FOOTER_REFRESH_GROUP_CLASS, "opacity-0")}>
                <Switch id="auto-refresh-loading" checked disabled className="scale-75" />
                <Label htmlFor="auto-refresh-loading" className="cursor-pointer whitespace-nowrap text-xs">
                  {t("toolbar.autoRefresh")}
                </Label>
              </div>

              <div className={SYSTEM_LOGS_FOOTER_OVERLAY_CLASS} aria-hidden="true">
                <div className={SYSTEM_LOGS_FOOTER_OVERLAY_STATUS_GROUP_CLASS}>
                  <Skeleton className="h-3 w-20 rounded-sm" />
                  <div className="h-3 w-px bg-border" />
                  <Skeleton className="h-3 w-28 rounded-sm" />
                  <div className="h-3 w-px bg-border" />
                  <Skeleton className="h-3 w-24 rounded-sm" />
                </div>

                <div className={SYSTEM_LOGS_FOOTER_OVERLAY_REFRESH_GROUP_CLASS}>
                  <Switch checked disabled tabIndex={-1} className="scale-75" />
                  <Skeleton className="h-3 w-20 rounded-sm" />
                </div>
              </div>
              </>}
            />
          </div>
        </div>
      </div>
    </div>
  )
}
