"use client";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Sheet, SheetContent, SheetHeader, SheetTitle, } from "@/components/ui/sheet";
import { Switch } from "@/components/ui/switch";
import { IconTerminal, } from "@/components/icons";
import { workbenchLogDrawerContentClassName } from "@/lib/ui/overlay-styles";
import { getStatusToneBgClass, getStatusToneTextClass } from "@/lib/status-config";
import { cn } from "@/lib/utils";
import { useAgentLogs } from "@/hooks/use-agent-logs";
import { filterStructuredLogLines, formatStructuredLogLine, StructuredLogViewer, type StructuredLogLevelFilter, } from "@/components/shared/visualization/structured-log-viewer";
import { TerminalLogToolbar } from "@/components/shared/visualization/terminal-log-toolbar";
import { LiveLogSurface } from "@/components/shared/visualization/terminal-log-surface";
import { TerminalLogCopyAllButton } from "@/components/shared/visualization/terminal-log-copy-all-button";
import type { Agent } from "@/types/agent.types";
import { getAgentConnectionIpDisplay } from "./agent-connection-ip";
const DEFAULT_LINES = 100;
const LOG_WINDOW_OPTIONS = [100, 200, 500, 1000, 2000, 5000] as const;
interface AgentLogDrawerProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    agentNode: Agent | null;
}
export function AgentLogDrawer({ open, onOpenChange, agentNode }: AgentLogDrawerProps) {
    const t = useTranslations("settings.agents");
    const container = "lunafox-agent";
    const connectionIpDisplay = agentNode
        ? getAgentConnectionIpDisplay(agentNode, {
            offline: t("connectionIp.offline"),
            unobserved: t("connectionIp.unobserved"),
        })
        : null;
    const [autoRefresh, setAutoRefresh] = useState(true);
    const [windowSize, setWindowSize] = useState<number>(DEFAULT_LINES);
    const { autoScroll, caughtUp, errorCode, errorMessage, gap, gapReason, handleViewportScroll, hasNewer, lines, phase, jumpToLatest, stopPolling, trimmedAfter, trimmedBefore, viewportRef, } = useAgentLogs({
        open,
        agentNode,
        container,
        windowSize,
        pollingEnabled: autoRefresh,
    });
    const [searchTerm, setSearchTerm] = useState("");
    const [levelFilter, setLevelFilter] = useState<StructuredLogLevelFilter>("all");
    const hasViewerError = phase === "error" || Boolean(errorCode || errorMessage);
    useEffect(() => {
        setWindowSize(DEFAULT_LINES);
    }, [agentNode?.id]);
    const filteredLines = useMemo(() => {
        return filterStructuredLogLines(lines, searchTerm, levelFilter);
    }, [lines, searchTerm, levelFilter]);
    const filteredLogText = useMemo(() => filteredLines.map((line) => formatStructuredLogLine(line, t("logs.truncated"))).join("\n"), [filteredLines, t]);
    const statusText = useMemo(() => {
        if (phase === "initialLoading")
            return t("logs.status.connecting");
        if (phase === "loadingOlder")
            return t("logs.status.loadingOlder");
        if (phase === "following")
            return t("logs.status.following");
        if (phase === "catchingUp")
            return t("logs.status.catchingUp");
        if (phase === "paused")
            return t("logs.status.scrollPaused");
        if (phase === "ready")
            return t("logs.status.ready");
        if (phase === "error")
            return t("logs.status.error");
        return t("logs.status.idle");
    }, [phase, t]);
    const statusDotClassName = useMemo(() => {
        if (hasViewerError)
            return getStatusToneBgClass("error");
        if (phase === "following")
            return getStatusToneBgClass("success");
        if (phase === "catchingUp" || phase === "initialLoading" || phase === "loadingOlder") {
            return getStatusToneBgClass("info");
        }
        if (phase === "paused")
            return getStatusToneBgClass("warning");
        return getStatusToneBgClass("muted");
    }, [hasViewerError, phase]);
    const statusTextClassName = useMemo(() => {
        if (hasViewerError)
            return getStatusToneTextClass("error");
        if (phase === "following")
            return getStatusToneTextClass("success");
        if (phase === "catchingUp" || phase === "initialLoading" || phase === "loadingOlder")
            return getStatusToneTextClass("info");
        if (phase === "paused")
            return getStatusToneTextClass("warning");
        return "text-muted-foreground";
    }, [hasViewerError, phase]);
    const resolveErrorMessage = useCallback((code: string | null, fallback: string) => {
        const normalized = code?.trim().toLowerCase();
        if (normalized === "container_not_found")
            return `[${code}] ${t("logs.errors.containerNotFound")}`;
        if (normalized === "loki_unavailable")
            return `[${code}] ${t("logs.errors.lokiUnavailable")}`;
        if (normalized === "query_timeout")
            return `[${code}] ${t("logs.errors.queryTimeout")}`;
        if (normalized === "agent_not_found")
            return `[${code}] ${t("logs.errors.agentNotFound")}`;
        if (normalized === "bad_request")
            return `[${code}] ${t("logs.errors.badRequest")}`;
        if (normalized === "internal_error")
            return `[${code}] ${t("logs.errors.unknown")}`;
        if (normalized === "agent_offline")
            return `[${code}] ${t("logs.errors.agentOffline")}`;
        if (fallback)
            return fallback;
        return t("logs.openFailed");
    }, [t]);
    const onToggleOpen = useCallback((nextOpen: boolean) => {
        if (!nextOpen) {
            stopPolling();
            setSearchTerm("");
            setLevelFilter("all");
            setWindowSize(DEFAULT_LINES);
            setAutoRefresh(true);
        }
        onOpenChange(nextOpen);
    }, [onOpenChange, stopPolling]);
    const onScroll = useCallback(() => {
        handleViewportScroll();
    }, [handleViewportScroll]);
    const statusDetail = useMemo(() => {
        if (hasViewerError) {
            return resolveErrorMessage(errorCode, errorMessage);
        }
        if (gap) {
            return t("logs.gapDetected");
        }
        if (!autoScroll) {
            return t("logs.pausedHint");
        }
        if (phase === "loadingOlder") {
            return t("logs.loadingOlderHint");
        }
        if (phase === "catchingUp") {
            return t("logs.catchingUpHint");
        }
        if (!caughtUp || hasNewer) {
            return t("logs.catchingUpHint");
        }
        return t("logs.ready");
    }, [autoScroll, caughtUp, errorCode, errorMessage, gap, hasNewer, hasViewerError, phase, resolveErrorMessage, t]);
    const isFiltering = searchTerm.trim() !== "" || levelFilter !== "all";
    const showJumpToLatest = phase !== "initialLoading" && (!autoScroll || hasNewer);
    return (<Sheet open={open} onOpenChange={onToggleOpen}>
      <SheetContent initialFocus={viewportRef} className={workbenchLogDrawerContentClassName}>
        <SheetHeader className="border-b px-4 py-2">
          <div className="flex gap-3 items-center">
            <div className="min-w-0">
              <SheetTitle className="flex gap-2 items-center">
                <IconTerminal className="h-4 w-4"/>
                {t("logs.title")}
              </SheetTitle>
              <p className="mt-0.5 text-muted-foreground text-xs truncate">
                {agentNode
            ? `${agentNode.name} (${connectionIpDisplay}) · ${container}`
            : t("logs.noAgent")}
              </p>
            </div>
          </div>
        </SheetHeader>

        <TerminalLogToolbar searchTerm={searchTerm} onSearchTermChange={setSearchTerm} searchPlaceholder={t("logs.searchPlaceholder")} levelFilter={levelFilter} onLevelFilterChange={setLevelFilter} lineWindow={{
          value: windowSize,
          options: LOG_WINDOW_OPTIONS,
          onValueChange: setWindowSize,
          label: t("logs.toolbar.lineWindow"),
        }} levelLabels={{
          all: t("logs.filterAll"),
          error: t("logs.filterError"),
          warn: t("logs.filterWarn"),
          info: t("logs.filterInfo"),
          debug: t("logs.filterDebug"),
        }}/>

        <LiveLogSurface
          viewportRef={viewportRef}
          onScroll={onScroll}
          focusable
          showJumpToLatest={showJumpToLatest}
          onJumpToLatest={jumpToLatest}
          jumpToLatestLabel={t("logs.jumpToLatest")}
          topRightAction={<TerminalLogCopyAllButton
            value={filteredLogText}
            copyLabel={t("logs.copyVisible")}
            copiedLabel={t("logs.copied")}
            toastId="agent-log-copy-all"
          />}
          footer={<>
            <div className="flex min-w-0 basis-full items-center gap-2 sm:basis-auto sm:gap-4">
              <span className="shrink-0 whitespace-nowrap">{lines.length} / {windowSize} {t("logs.toolbar.linesUnit")}</span>
              <Separator orientation="vertical" className="hidden h-3 sm:block"/>
              <span className="shrink-0 whitespace-nowrap">{t("logs.toolbar.agentContainerSource", { container })}</span>
              <Separator orientation="vertical" className="hidden h-3 sm:block"/>
              <span className="flex min-w-0 flex-1 gap-1.5 items-center">
                <span className={cn("h-2 w-2 shrink-0 rounded-full", statusDotClassName)} aria-hidden="true"/>
                <span className={cn("shrink-0 text-[11px] font-medium", statusTextClassName)}>
                  {statusText}
                </span>
                <span className="truncate">
                  {statusDetail}
                </span>
              </span>
            </div>
            <div className="flex shrink-0 gap-2 items-center whitespace-nowrap">
              <Switch id="agent-log-auto-refresh" checked={autoRefresh} onCheckedChange={setAutoRefresh} className="shrink-0 scale-75"/>
              <Label htmlFor="agent-log-auto-refresh" className="cursor-pointer whitespace-nowrap text-xs">
                {t("logs.toolbar.autoRefresh")}
              </Label>
            </div>
          </>}
        >
            {trimmedAfter && (<div className="mb-3 text-[11px] text-warning/90">{t("logs.trimmedNewerHint")}</div>)}
            {filteredLines.length === 0 ? (<div className="text-muted-foreground">
                {isFiltering ? t("logs.noMatch") : t("logs.empty")}
              </div>) : (<>
                {trimmedBefore && (<div className="mb-2 text-[11px] text-warning/90">
                    {t("logs.lineLimitHint", { max: windowSize })}
                  </div>)}
                {gap && (<div className="mb-2 text-[11px] text-warning/90">
                    {gapReason ? `${t("logs.gapDetected")} (${gapReason})` : t("logs.gapDetected")}
                  </div>)}
                <StructuredLogViewer
                  viewportRef={viewportRef}
                  lines={filteredLines}
                  empty={isFiltering ? t("logs.noMatch") : t("logs.empty")}
                  truncatedLabel={t("logs.truncated")}
                />
              </>)}
        </LiveLogSurface>
      </SheetContent>
    </Sheet>);
}
