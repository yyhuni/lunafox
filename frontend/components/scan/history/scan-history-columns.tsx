"use client";
import React from "react";
import { ColumnDef } from "@tanstack/react-table";
import Link from "next/link";
import { useTranslations } from "next-intl";
import { Checkbox } from "@/components/ui/checkbox";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { DropdownMenuItem, DropdownMenuSeparator, } from "@/components/ui/dropdown-menu";
import type { ScanRecord, ScanStatus, ScanTriggerType } from "@/types/scan.types";
import { Eye, semanticIcons, Trash2 } from "@/components/icons";
import { DataTableColumnHeader } from "@/components/shared/data-table/column-header";
import { DenseRowActionMenu } from "@/components/shared/data-table/menu-owners";
import { TimestampCell } from "@/components/shared/data-table/timestamp-cell";
import { ScanStatusBadge } from "@/components/scan/scan-status-badge";
import { textRole } from "@/lib/typography";
import { cn } from "@/lib/utils";
import { scanHistoryTableColumnLayout } from "./scan-history-table-layout";
// Translation type definitions
export interface ScanHistoryTranslations {
    columns: {
        target: string;
        summary: string;
        executedEngines: string;
        triggerType: string;
        createdAt: string;
        status: string;
        progress: string;
    };
    actions: {
        scanDetail: string;
        runtimeDetail: string;
        openMenu: string;
        stop: string;
        stopScanPending: string;
        delete: string;
        selectAll: string;
        selectRow: string;
    };
    tooltips: {
        viewProgress: string;
    };
    status: {
        cancelled: string;
        succeeded: string;
        failed: string;
        pending: string;
        running: string;
    };
    summary: {
        subdomains: string;
        websites: string;
        ipAddresses: string;
        endpoints: string;
        vulnerabilities: string;
    };
    triggerTypes: {
        manual: string;
        scheduled: string;
        ai: string;
    };
}
// StatusBadge component removed in favor of ScanStatusBadge in separate file
// function StatusBadge({ ... }) { ... }
// Column creation function parameter types
interface CreateColumnsProps {
    formatDate: (dateString: string) => string;
    handleDelete: (scan: ScanRecord) => void;
    handleStop: (scan: ScanRecord) => void;
    handleStatusClick?: (scan: ScanRecord) => void;
    statusActionLabel?: string;
    statusClickable?: boolean;
    t: ScanHistoryTranslations;
    hideTargetColumn?: boolean;
    executedEngineNamesByScanId?: ReadonlyMap<number, string[]>;
    executedEngineDescriptionsByScanId?: ReadonlyMap<number, string[]>;
}
type ScanHistorySummaryBadge = {
    id: string;
    count: number;
    name: string;
    type: "subdomain" | "website" | "ip" | "endpoint" | "vulnerability";
};
// Loading retains the dense row baseline; resolved summary and engine groups
// wrap complete Badge items so scan context is never hidden by the column edge.
const SCAN_HISTORY_SUMMARY_BADGE_LIST_CLASS = "flex min-w-0 flex-wrap items-center gap-1";
const SCAN_HISTORY_ENGINE_BADGE_LIST_CLASS = "flex min-w-0 flex-wrap items-center gap-1";
const SCAN_HISTORY_ENGINE_TOOLTIP_DELAY_MS = 400;
const SCAN_HISTORY_ENGINE_TOOLTIP_CLOSE_DELAY_MS = 100;
const SCAN_HISTORY_STATUS_TOOLTIP_DELAY_MS = 300;
const SCAN_HISTORY_STATUS_TOOLTIP_CLOSE_DELAY_MS = 100;
const SCAN_HISTORY_TRIGGER_TOOLTIP_DELAY_MS = 300;
const SCAN_HISTORY_TRIGGER_TOOLTIP_CLOSE_DELAY_MS = 100;

// The selection cell and target cell each contribute to the leading gutter; use
// the target cell's padding so the provenance cue stays attached to row identity.
function ScanTriggerSourceCell({
    triggerType,
    label,
    columnLabel,
}: {
    triggerType: ScanTriggerType;
    label: string;
    columnLabel: string;
}) {
    const TriggerIcon = semanticIcons.triggerSource[triggerType];
    const accessibleLabel = `${columnLabel}: ${label}`;

    return (<TooltipProvider delay={SCAN_HISTORY_TRIGGER_TOOLTIP_DELAY_MS} closeDelay={SCAN_HISTORY_TRIGGER_TOOLTIP_CLOSE_DELAY_MS}>
      <Tooltip>
        <TooltipTrigger
          closeOnClick={false}
          render={<span
            className="radius-control-subtle -ml-2 inline-flex size-6 shrink-0 cursor-help items-center justify-center text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:ring-offset-1"
            role="img"
            tabIndex={0}
            aria-label={accessibleLabel}
            title={label}
            data-row-click-exempt="true"
            data-trigger-type={triggerType}
          />}
        >
          <TriggerIcon aria-hidden="true" className="size-4" />
        </TooltipTrigger>
        <TooltipContent>{label}</TooltipContent>
      </Tooltip>
    </TooltipProvider>);
}

function EmptySummaryBadge() {
    const tOverview = useTranslations("scan.history.overview");
    return (<Badge variant="outline" data-badge-type="empty">
      -
      <span className="sr-only">{tOverview("noSummary")}</span>
    </Badge>);
}
/**
 * Create scan history table column definitions
 */
export const createScanHistoryColumns = ({ formatDate, handleDelete, handleStop, handleStatusClick, statusActionLabel, statusClickable = true, t, hideTargetColumn = false, executedEngineNamesByScanId = new Map(), executedEngineDescriptionsByScanId = new Map(), }: CreateColumnsProps): ColumnDef<ScanRecord>[] => {
    const columns: ColumnDef<ScanRecord>[] = [
        {
            id: "select",
            size: scanHistoryTableColumnLayout.select.size,
            minSize: scanHistoryTableColumnLayout.select.minSize,
            maxSize: scanHistoryTableColumnLayout.select.maxSize,
            enableResizing: false,
            header: ({ table }) => (<Checkbox checked={table.getIsAllPageRowsSelected()} indeterminate={!table.getIsAllPageRowsSelected() && table.getIsSomePageRowsSelected()} onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)} aria-label={t.actions.selectAll}/>),
            cell: ({ row }) => (<Checkbox checked={row.getIsSelected()} onCheckedChange={(value) => row.toggleSelected(!!value)} aria-label={t.actions.selectRow}/>),
            enableSorting: false,
            enableHiding: false,
        },
        {
            accessorKey: "target",
            accessorFn: (row) => row.target?.displayName,
            size: scanHistoryTableColumnLayout.target.size,
            minSize: scanHistoryTableColumnLayout.target.minSize,
            maxSize: scanHistoryTableColumnLayout.target.maxSize,
            meta: {
                title: t.columns.target,
                widthPolicy: { mode: "flex", flex: 1.1 },
            },
            header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.target}/>),
            cell: ({ row }) => {
                const targetName = row.original.target?.displayName;
                const targetId = row.original.targetId;
                return (<div className="flex min-w-0 items-center gap-2">
            <ScanTriggerSourceCell
              triggerType={row.original.triggerType}
              label={t.triggerTypes[row.original.triggerType]}
              columnLabel={t.columns.triggerType}
            />
            {targetId ? (<Link href={`/targets/${targetId}/overview/`} className={cn("min-w-0 truncate text-left transition-colors underline-offset-2 hover:text-primary hover:underline", textRole.tableCellPrimary)} title={targetName}>
                {targetName}
              </Link>) : (<span className={cn("min-w-0 truncate", textRole.tableCellPrimary)} title={targetName}>
                {targetName}
              </span>)}
          </div>);
            },
        },
        {
            accessorKey: "cachedStats",
            accessorFn: (row) => row.cachedStats,
            meta: {
                title: t.columns.summary,
                widthPolicy: { mode: "flex", flex: 1.3 },
            },
            header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.summary}/>),
            size: scanHistoryTableColumnLayout.cachedStats.size,
            minSize: scanHistoryTableColumnLayout.cachedStats.minSize,
            maxSize: scanHistoryTableColumnLayout.cachedStats.maxSize,
            cell: ({ row }) => {
                const subdomains = row.original.cachedStats?.subdomainsCount ?? 0;
                const websites = row.original.cachedStats?.websitesCount ?? 0;
                const endpoints = row.original.cachedStats?.endpointsCount ?? 0;
                const ips = row.original.cachedStats?.ipsCount ?? 0;
                const vulns = row.original.cachedStats?.vulnsTotal ?? 0;
                const badges: ScanHistorySummaryBadge[] = [];
                if (subdomains > 0) {
                    badges.push({
                        id: "subdomains",
                        count: subdomains,
                        name: t.summary.subdomains,
                        type: "subdomain",
                    });
                }
                if (websites > 0) {
                    badges.push({
                        id: "websites",
                        count: websites,
                        name: t.summary.websites,
                        type: "website",
                    });
                }
                if (ips > 0) {
                    badges.push({
                        id: "ips",
                        count: ips,
                        name: t.summary.ipAddresses,
                        type: "ip",
                    });
                }
                if (endpoints > 0) {
                    badges.push({
                        id: "endpoints",
                        count: endpoints,
                        name: t.summary.endpoints,
                        type: "endpoint",
                    });
                }
                if (vulns > 0) {
                    badges.push({
                        id: "vulnerabilities",
                        count: vulns,
                        name: t.summary.vulnerabilities,
                        type: "vulnerability",
                    });
                }
                const triggerSource = hideTargetColumn ? (<ScanTriggerSourceCell
                  triggerType={row.original.triggerType}
                  label={t.triggerTypes[row.original.triggerType]}
                  columnLabel={t.columns.triggerType}
                />) : null;
                if (badges.length === 0) {
                    return (<div className="flex min-w-0 items-center gap-2">
              {triggerSource}
              <EmptySummaryBadge />
            </div>);
                }
                return (<div className="flex min-w-0 items-center gap-2">
            {triggerSource}
            <div data-scan-summary-badges className={cn(SCAN_HISTORY_SUMMARY_BADGE_LIST_CLASS, "min-w-0")} title={badges.map((badge) => `${badge.count} ${badge.name}`).join(", ")}>
            {badges.map((badge) => (<Badge key={badge.id} variant="outline" data-badge-type={badge.type} className={cn(textRole.badgeSubtle, "shrink-0")} title={`${badge.count} ${badge.name}`}>
                {badge.count} {badge.name}
              </Badge>))}
          </div>
          </div>);
            },
            enableSorting: false,
        },
        {
            id: "executedEngines",
            accessorFn: (row) => executedEngineNamesByScanId.get(row.id) ?? [],
            size: scanHistoryTableColumnLayout.executedEngines.size,
            minSize: scanHistoryTableColumnLayout.executedEngines.minSize,
            maxSize: scanHistoryTableColumnLayout.executedEngines.maxSize,
            enableResizing: false,
            meta: {
                title: t.columns.executedEngines,
                enableAutoSize: false,
                widthPolicy: { mode: "flex", flex: 1, fill: true },
            },
            header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.executedEngines}/>),
            cell: ({ row }) => {
                const executedEngineNames = executedEngineNamesByScanId.get(row.original.id) ?? [];
                const executedEngineDescriptions = executedEngineDescriptionsByScanId.get(row.original.id) ?? [];
                if (!executedEngineNames || executedEngineNames.length === 0) {
                    return (<Badge variant="outline" data-badge-type="empty">
              -
            </Badge>);
                }
                return (<TooltipProvider delay={SCAN_HISTORY_ENGINE_TOOLTIP_DELAY_MS} closeDelay={SCAN_HISTORY_ENGINE_TOOLTIP_CLOSE_DELAY_MS}>
                  <div data-scan-engine-badges className={SCAN_HISTORY_ENGINE_BADGE_LIST_CLASS} title={executedEngineNames.join(", ")}>
              {executedEngineNames.map((name, index) => {
                const description = executedEngineDescriptions[index];
                const badge = (<Badge variant="secondary" data-badge-type="engine" className={cn(textRole.badgeSubtle, "max-w-full shrink-0 truncate")} title={name}>
                  {name}
                </Badge>);

                if (!description) {
                    return <React.Fragment key={`${name}-${index}`}>{badge}</React.Fragment>;
                }

                return (<Tooltip key={`${name}-${index}`}>
                  <TooltipTrigger closeOnClick={false} render={<Badge render={<span tabIndex={0} />} variant="secondary" data-badge-type="engine" className={cn(textRole.badgeSubtle, "max-w-full shrink-0 truncate")} title={name}/> }>
                    {name}
                  </TooltipTrigger>
                  <TooltipContent className="max-w-sm whitespace-normal text-left">
                    {description}
                  </TooltipContent>
                </Tooltip>);
              })}
            </div>
          </TooltipProvider>);
            },
        },
        {
            accessorKey: "createdAt",
            size: scanHistoryTableColumnLayout.createdAt.size,
            minSize: scanHistoryTableColumnLayout.createdAt.minSize,
            maxSize: scanHistoryTableColumnLayout.createdAt.maxSize,
            enableResizing: false,
            meta: {
                title: t.columns.createdAt,
                orderBy: "createdAt",
                serverSortPerformance: "indexed",
            },
            header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.createdAt}/>),
            cell: ({ row }) => {
                const createdAt = row.getValue("createdAt") as string;
                return <TimestampCell value={formatDate(createdAt)} className="truncate"/>;
            },
        },
        {
            accessorKey: "status",
            size: scanHistoryTableColumnLayout.status.size,
            minSize: scanHistoryTableColumnLayout.status.minSize,
            maxSize: scanHistoryTableColumnLayout.status.maxSize,
            enableResizing: false,
            meta: { title: t.columns.status },
            header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.status}/>),
            cell: ({ row }) => {
                const status = row.getValue("status") as ScanStatus;
                const progress = row.original.progress;
                const isClickable = Boolean(handleStatusClick) && statusClickable;
                if (isClickable) {
                    const runtimeDetailLabel = statusActionLabel ?? t.actions.runtimeDetail;
                    return (<TooltipProvider delay={SCAN_HISTORY_STATUS_TOOLTIP_DELAY_MS} closeDelay={SCAN_HISTORY_STATUS_TOOLTIP_CLOSE_DELAY_MS}>
                      <Tooltip>
                        <TooltipTrigger closeOnClick={false} render={<button type="button" onClick={(event) => {
                            event.stopPropagation();
                            handleStatusClick?.(row.original);
                        }} className={cn("w-full cursor-pointer rounded-md text-left transition-[background-color,box-shadow]", "hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:ring-offset-1")} aria-label={runtimeDetailLabel} data-row-click-exempt/>}>
                          <ScanStatusBadge status={status} progress={progress} labels={t.status} variant="inline" // Using F2 variant (Inline Block)
                              />
                        </TooltipTrigger>
                        <TooltipContent>{runtimeDetailLabel}</TooltipContent>
                      </Tooltip>
                    </TooltipProvider>);
                }
                return (<div className={cn("cursor-default")}>
            <ScanStatusBadge status={status} progress={progress} labels={t.status} variant="inline" // Using F2 variant (Inline Block)
                />
          </div>);
            },
        },
        // Progress column removed as it's integrated into status
        // {
        //   accessorKey: "progress", 
        //   ... 
        // },
        {
            id: "actions",
            size: scanHistoryTableColumnLayout.actions.size,
            minSize: scanHistoryTableColumnLayout.actions.minSize,
            maxSize: scanHistoryTableColumnLayout.actions.maxSize,
            enableResizing: false,
            cell: ({ row }) => {
                const scan = row.original;
                const canStop = scan.status === 'running' || scan.status === 'pending';
                return (<DenseRowActionMenu ariaLabel={t.actions.openMenu}>
            <DropdownMenuItem render={<Link href={`/scan/history/${scan.id}/overview/`}/>}>
                <Eye />
                {t.actions.scanDetail}
              </DropdownMenuItem>
            {canStop && (<>
                <DropdownMenuItem onClick={() => handleStop(scan)}>
                  <semanticIcons.action.stop />
                  {t.actions.stop}
                </DropdownMenuItem>
                <DropdownMenuSeparator />
              </>)}
            <DropdownMenuItem onClick={() => handleDelete(scan)} variant="destructive">
              <Trash2 />
              {t.actions.delete}
            </DropdownMenuItem>
          </DenseRowActionMenu>);
            },
            enableSorting: false,
            enableHiding: false,
        },
    ];
    // Filter out target column if hideTargetColumn is true
    if (hideTargetColumn) {
        return columns.filter((col) => (col as {
            accessorKey?: string;
        }).accessorKey !== 'target');
    }
    return columns;
};
