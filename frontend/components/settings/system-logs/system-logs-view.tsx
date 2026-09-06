"use client";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { PageHeader } from "@/components/common/page-header";
import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { filterStructuredLogLines, formatStructuredLogLine, StructuredLogViewer, type StructuredLogLevelFilter, } from "@/components/shared/visualization/structured-log-viewer";
import { TerminalLogToolbar } from "@/components/shared/visualization/terminal-log-toolbar";
import { LiveLogSurface } from "@/components/shared/visualization/terminal-log-surface";
import { TerminalLogCopyAllButton } from "@/components/shared/visualization/terminal-log-copy-all-button";
import { SYSTEM_LOGS_CONTENT_SHELL_CLASS, SYSTEM_LOGS_DEFAULT_LINES, SYSTEM_LOGS_FOOTER_REFRESH_GROUP_CLASS, SYSTEM_LOGS_FOOTER_STATUS_GROUP_CLASS, SYSTEM_LOGS_HEADER_SHELL_CLASS, SYSTEM_LOGS_PAGE_SHELL_CLASS, SYSTEM_LOGS_TERMINAL_SHELL_CLASS, } from "@/components/settings/system-logs/system-logs-layout";
import { useSystemLogs } from "@/hooks/use-system-logs";
import { getStatusToneBgClass, getStatusToneTextClass } from "@/lib/status-config";
import { cn } from "@/lib/utils";
export interface SystemLogsViewProps {
    pageTitle: string;
    pageDescription: string;
    onReady?: () => void;
    deferInitialSkeleton?: boolean;
}
const LOG_WINDOW_OPTIONS = [100, 200, 500, 1000, 2000, 5000] as const;
export function SystemLogsView({ pageTitle, pageDescription, onReady, deferInitialSkeleton = false, }: SystemLogsViewProps) {
    const t = useTranslations("settings.systemLogs");
    const [windowSize, setWindowSize] = useState<number>(SYSTEM_LOGS_DEFAULT_LINES);
    const [autoRefresh, setAutoRefresh] = useState(true);
    const [searchTerm, setSearchTerm] = useState("");
    const [levelFilter, setLevelFilter] = useState<StructuredLogLevelFilter>("all");
    const { autoScroll, caughtUp, errorCode, errorMessage, gap, gapReason, handleViewportScroll, hasNewer, isLoading, lines: logLines, phase, jumpToLatest, trimmedAfter, trimmedBefore, viewportRef, } = useSystemLogs({
        windowSize,
        autoRefresh,
    });
    const isInitialLoading = isLoading && logLines.length === 0;
    const hasViewerError = phase === "error" || Boolean(errorCode || errorMessage);
    const filteredLines = useMemo(() => {
        return filterStructuredLogLines(logLines, searchTerm, levelFilter);
    }, [levelFilter, logLines, searchTerm]);
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
        if (normalized === "loki_unavailable")
            return `[${code}] ${t("logs.errors.lokiUnavailable")}`;
        if (normalized === "query_timeout")
            return `[${code}] ${t("logs.errors.queryTimeout")}`;
        if (normalized === "bad_request")
            return `[${code}] ${t("logs.errors.badRequest")}`;
        if (normalized === "internal_error")
            return `[${code}] ${t("logs.errors.unknown")}`;
        if (fallback)
            return fallback;
        return t("logs.openFailed");
    }, [t]);
    const statusDetail = useMemo(() => {
        if (hasViewerError) {
            return resolveErrorMessage(errorCode, errorMessage);
        }
        if (gap) {
            return t("logs.gapDetected");
        }
        if (phase === "paused") {
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
    }, [caughtUp, errorCode, errorMessage, gap, hasNewer, hasViewerError, phase, resolveErrorMessage, t]);
    const onScroll = useCallback(() => {
        handleViewportScroll();
    }, [handleViewportScroll]);
    useEffect(() => {
        // `idle` is the hook's pre-request render, not a settled first frame.
        // Signalling the route boundary here can cancel the very request that
        // establishes the content geometry.
        if (phase === "idle" || isInitialLoading)
            return;
        let secondFrame: number | null = null;
        const first = window.requestAnimationFrame(() => {
            secondFrame = window.requestAnimationFrame(() => {
                onReady?.();
            });
        });
        return () => {
            window.cancelAnimationFrame(first);
            if (secondFrame !== null) {
                window.cancelAnimationFrame(secondFrame);
            }
        };
    }, [isInitialLoading, onReady, phase]);
    const isFiltering = searchTerm.trim() !== "" || levelFilter !== "all";
    const showJumpToLatest = !isInitialLoading && (!autoScroll || hasNewer);
    // The route boundary hides this branch while it owns the first-screen skeleton.
    // Keep the log viewer mounted so its initial request can settle and release that boundary.
    return (<div className={SYSTEM_LOGS_PAGE_SHELL_CLASS} data-loading-hidden-readiness={deferInitialSkeleton ? "true" : undefined}>
      <div {...getLoadingStructureSlotAttributes("system-logs-header")} className={SYSTEM_LOGS_HEADER_SHELL_CLASS}>
        <PageHeader code="LOG-01" title={pageTitle} description={pageDescription}/>
      </div>

      <div className={SYSTEM_LOGS_CONTENT_SHELL_CLASS}>
        <div className={SYSTEM_LOGS_TERMINAL_SHELL_CLASS}>
          <div {...getLoadingStructureSlotAttributes("system-logs-terminal-toolbar")} className="shrink-0">
            <TerminalLogToolbar searchTerm={searchTerm} onSearchTermChange={setSearchTerm} searchPlaceholder={t("logs.searchPlaceholder")} levelFilter={levelFilter} onLevelFilterChange={setLevelFilter} lineWindow={{
            value: windowSize,
            options: LOG_WINDOW_OPTIONS,
            onValueChange: setWindowSize,
            label: t("toolbar.lineWindow"),
          }} levelLabels={{
            all: t("logs.filterAll"),
            error: t("logs.filterError"),
            warn: t("logs.filterWarn"),
            info: t("logs.filterInfo"),
            debug: t("logs.filterDebug"),
          }}/>
          </div>

          <div {...getLoadingStructureSlotAttributes("system-logs-log-surface")} className="flex min-h-0 flex-1 flex-col">
            <LiveLogSurface
            viewportRef={viewportRef}
            onScroll={onScroll}
            showJumpToLatest={showJumpToLatest}
            onJumpToLatest={jumpToLatest}
            jumpToLatestLabel={t("logs.jumpToLatest")}
            topRightAction={<TerminalLogCopyAllButton
              value={filteredLogText}
              copyLabel={t("logs.copyVisible")}
              copiedLabel={t("logs.copied")}
              toastId="system-log-copy-all"
            />}
            footer={<>
              <div className={SYSTEM_LOGS_FOOTER_STATUS_GROUP_CLASS}>
                <span className="shrink-0 whitespace-nowrap">{logLines.length} / {windowSize} {t("toolbar.linesUnit")}</span>
                <Separator orientation="vertical" className="hidden h-3 sm:block"/>
                <span className="shrink-0 whitespace-nowrap">{t("toolbar.serverSource")}</span>
                <Separator orientation="vertical" className="hidden h-3 sm:block"/>
                <span className="flex min-w-0 flex-1 gap-1.5 items-center">
                  <span className={cn("h-2 w-2 shrink-0 rounded-full", statusDotClassName)} aria-hidden="true"/>
                  <span className={cn("shrink-0 text-[11px] font-medium", statusTextClassName)}>
                    {statusText}
                  </span>
                  <span className="truncate">{statusDetail}</span>
                </span>
              </div>
              <div className={SYSTEM_LOGS_FOOTER_REFRESH_GROUP_CLASS}>
                <Switch id="auto-refresh" checked={autoRefresh} onCheckedChange={setAutoRefresh} className="shrink-0 scale-75"/>
                <Label htmlFor="auto-refresh" className="cursor-pointer whitespace-nowrap text-xs">
                  {t("toolbar.autoRefresh")}
                </Label>
              </div>
            </>}
          >
              {trimmedAfter && (<div className="mb-3 text-[11px] text-warning/90">{t("logs.trimmedNewerHint")}</div>)}
              {trimmedBefore && (<div className="mb-2 text-[11px] text-warning/90">
                  {t("logs.lineLimitHint", { max: windowSize })}
                </div>)}
              {gap && (<div className="mb-2 text-[11px] text-warning/90">
                  {gapReason ? `${t("logs.gapDetected")} (${gapReason})` : t("logs.gapDetected")}
                </div>)}
              <StructuredLogViewer viewportRef={viewportRef} lines={filteredLines} empty={isFiltering ? t("logs.noMatch") : t("logs.empty")} truncatedLabel={t("logs.truncated")}/>
            </LiveLogSurface>
          </div>
        </div>
      </div>
    </div>);
}
