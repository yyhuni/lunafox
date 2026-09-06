"use client";
import { ColumnDef } from "@tanstack/react-table";
import { Circle, CheckCircle2 } from "@/components/icons";
import { Checkbox } from "@/components/ui/checkbox";
import { Badge } from "@/components/ui/badge";
import { ExpandableUrlCell } from "@/components/shared/data-table/expandable-cell";
import { DataTableColumnHeader } from "@/components/shared/data-table/column-header";
import { TimestampCell } from "@/components/shared/data-table/timestamp-cell";
import { getSeverityVariant } from "@/lib/severity-config";
import { getStatusToneTextClass } from "@/lib/status-config";
import { cn } from "@/lib/utils";
import { vulnerabilitiesTableColumnLayout } from "./vulnerabilities-table-layout";
import type { Vulnerability, VulnerabilitySeverity } from "@/types/vulnerability.types";
// Translation type definitions
export interface VulnerabilityTranslations {
    columns: {
        status: string;
        severity: string;
        source: string;
        vulnType: string;
        url: string;
        createdAt: string;
    };
    actions: {
        selectAll: string;
        selectRow: string;
    };
    tooltips: {
        vulnDetails: string;
        reviewed: string;
        pending: string;
    };
    severity: {
        critical: string;
        high: string;
        medium: string;
        low: string;
        info: string;
    };
}
interface ColumnActions {
    formatDate: (date: string) => string;
    t: VulnerabilityTranslations;
    includeSelection?: boolean;
}
export function createVulnerabilityColumns({ formatDate, t, includeSelection = true }: ColumnActions): ColumnDef<Vulnerability>[] {
    function getReviewStatusIndicatorClassName(isPending: boolean): string {
        return cn("inline-flex size-6 items-center justify-center", getStatusToneTextClass(isPending ? "muted" : "success"));
    }
    const selectionColumn: ColumnDef<Vulnerability> = {
            id: "select",
            size: vulnerabilitiesTableColumnLayout.select.size,
            minSize: vulnerabilitiesTableColumnLayout.select.minSize,
            maxSize: vulnerabilitiesTableColumnLayout.select.maxSize,
            enableResizing: false,
            header: ({ table }) => (<Checkbox checked={table.getIsAllPageRowsSelected()} indeterminate={!table.getIsAllPageRowsSelected() && table.getIsSomePageRowsSelected()} onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)} aria-label={t.actions.selectAll}/>),
            cell: ({ row }) => (<Checkbox checked={row.getIsSelected()} onCheckedChange={(value) => row.toggleSelected(!!value)} aria-label={t.actions.selectRow}/>),
            enableSorting: false,
            enableHiding: false,
        }
    const dataColumns: ColumnDef<Vulnerability>[] = [
        {
            id: "reviewStatus",
            meta: { title: t.columns.status },
            size: vulnerabilitiesTableColumnLayout.reviewStatus.size,
            minSize: vulnerabilitiesTableColumnLayout.reviewStatus.minSize,
            maxSize: vulnerabilitiesTableColumnLayout.reviewStatus.maxSize,
            enableResizing: false,
            header: t.columns.status,
            cell: ({ row }) => {
                const isReviewed = row.original.isReviewed;
                const isPending = !isReviewed;
                return (<span aria-label={isPending ? t.tooltips.pending : t.tooltips.reviewed} className={getReviewStatusIndicatorClassName(isPending)} role="img">
            {isPending ? (<Circle className="h-3.5 w-3.5"/>) : (<CheckCircle2 className="h-3.5 w-3.5"/>)}
          </span>);
            },
            enableSorting: false,
            enableHiding: false,
        },
        {
            accessorKey: "severity",
            meta: { title: t.columns.severity },
            header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.severity}/>),
            size: vulnerabilitiesTableColumnLayout.severity.size,
            minSize: vulnerabilitiesTableColumnLayout.severity.minSize,
            maxSize: vulnerabilitiesTableColumnLayout.severity.maxSize,
            enableResizing: false,
            cell: ({ row }) => {
                const severity = row.getValue("severity") as VulnerabilitySeverity;
                return (<Badge variant={getSeverityVariant(severity)}>
            {t.severity[severity]}
          </Badge>);
            },
        },
        {
            accessorKey: "source",
            meta: { title: t.columns.source },
            header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.source}/>),
            size: vulnerabilitiesTableColumnLayout.source.size,
            minSize: vulnerabilitiesTableColumnLayout.source.minSize,
            maxSize: vulnerabilitiesTableColumnLayout.source.maxSize,
            enableResizing: false,
            cell: ({ row }) => {
                const source = row.getValue("source") as string;
                return (<Badge variant="outline">
            {source}
          </Badge>);
            },
        },
        {
            accessorKey: "vulnType",
            meta: { title: t.columns.vulnType },
            header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.vulnType}/>),
            size: vulnerabilitiesTableColumnLayout.vulnType.size,
            minSize: vulnerabilitiesTableColumnLayout.vulnType.minSize,
            maxSize: vulnerabilitiesTableColumnLayout.vulnType.maxSize,
            cell: ({ row }) => {
                const vulnType = row.getValue("vulnType") as string;
                return vulnType;
            },
        },
        {
            accessorKey: "url",
            meta: { title: t.columns.url },
            header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.url}/>),
            size: vulnerabilitiesTableColumnLayout.url.size,
            minSize: vulnerabilitiesTableColumnLayout.url.minSize,
            maxSize: vulnerabilitiesTableColumnLayout.url.maxSize,
            cell: ({ row }) => (<ExpandableUrlCell value={row.original.url} textRoleName="tableCellPrimary"/>),
        },
        {
            accessorKey: "createdAt",
            meta: {
                title: t.columns.createdAt,
                orderBy: "createdAt",
                serverSortPerformance: "idx_vuln_created_at_id / idx_vuln_target_created_at_id / idx_vuln_snap_scan_created_at_id",
                firstSortDirection: "desc",
            },
            header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.createdAt}/>),
            size: vulnerabilitiesTableColumnLayout.createdAt.size,
            minSize: vulnerabilitiesTableColumnLayout.createdAt.minSize,
            maxSize: vulnerabilitiesTableColumnLayout.createdAt.maxSize,
            enableResizing: false,
            cell: ({ row }) => {
                const createdAt = row.getValue("createdAt") as string;
                return <TimestampCell value={formatDate(createdAt)}/>;
            },
        },
    ];
    return includeSelection ? [selectionColumn, ...dataColumns] : dataColumns;
}
