"use client";
import React from "react";
import { ColumnDef } from "@tanstack/react-table";
import Link from "next/link";
import { Checkbox } from "@/components/ui/checkbox";
import { DropdownMenuItem, DropdownMenuSeparator, } from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger, } from "@/components/ui/tooltip";
import { semanticIcons } from "@/components/icons";
import { DataTableColumnHeader } from "@/components/shared/data-table/column-header";
import { ExpandableBadgeList } from "@/components/shared/data-table/expandable-cell";
import { DenseRowActionMenu } from "@/components/shared/data-table/menu-owners";
import { DenseRowActionButton } from "@/components/shared/data-table/row-actions";
import { QuietCopyButton } from "@/components/shared/data-table/row-actions";
import { TimestampCell } from "@/components/shared/data-table/timestamp-cell";
import { textRole } from "@/lib/typography";
import { cn } from "@/lib/utils";
import type { Target, TargetType } from "@/types/target.types";
import { allTargetsTableColumnLayout } from "./all-targets-table-layout";
const TARGET_NAME_COPY_TOAST_ID = "target-name-copy";
const ViewIcon = semanticIcons.action.view;
const RunIcon = semanticIcons.action.run;
const ScheduleIcon = semanticIcons.action.schedule;
const DeleteIcon = semanticIcons.action.delete;
// Translation type definitions
export interface AllTargetsTranslations {
    columns: {
        target: string;
        organization: string;
        addedOn: string;
        lastScanned: string;
        actions: string;
    };
    actions: {
        scheduleScan: string;
        delete: string;
        selectAll: string;
        selectRow: string;
        openMenu: string;
    };
    tooltips: {
        targetDetails: string;
        targetSummary: string;
        initiateScan: string;
        clickToCopy: string;
        copied: string;
    };
    targetTypes: Record<TargetType, string>;
}
interface CreateColumnsProps {
    formatDate: (dateString: string) => string;
    navigate: (path: string) => void;
    handleDelete: (target: Target) => void;
    handleInitiateScan: (target: Target) => void;
    handleScheduleScan: (target: Target) => void;
    t: AllTargetsTranslations;
}
/**
 * Target name cell component
 */
const TargetNameCell = React.memo(function TargetNameCell({ name, targetId, targetType, t, }: {
    name: string;
    targetId: number;
    targetType: TargetType;
    t: AllTargetsTranslations;
}) {
    const TargetTypeIcon = semanticIcons.concept[targetType];
    const targetTypeLabel = t.targetTypes[targetType];

    return (<div className="group flex min-w-0 items-center gap-2">
      <TargetTypeIcon title={targetTypeLabel} role="img" aria-label={targetTypeLabel} className="size-4 shrink-0 text-muted-foreground transition-colors group-hover:text-foreground group-data-[state=selected]:text-foreground" />
      <div className="flex min-w-0 max-w-full flex-1 items-center gap-1">
        <Tooltip>
          <TooltipTrigger render={<Link href={`/targets/${targetId}/overview/`} className={cn("block min-w-0 truncate text-left transition-colors underline-offset-2 hover:text-primary hover:underline", textRole.tableCellPrimary)}/> }>
            {name}
          </TooltipTrigger>
          <TooltipContent>{t.tooltips.targetDetails}</TooltipContent>
        </Tooltip>
        <QuietCopyButton value={name} copyLabel={t.tooltips.clickToCopy} copiedLabel={t.tooltips.copied} toastId={TARGET_NAME_COPY_TOAST_ID}/>
      </div>
    </div>);
});
/**
 * Target row actions component
 */
const TargetRowActions = React.memo(function TargetRowActions({ onView, onInitiateScan, onScheduleScan, onDelete, t, }: {
    onView: () => void;
    onInitiateScan: () => void;
    onScheduleScan: () => void;
    onDelete: () => void;
    t: AllTargetsTranslations;
}) {
    return (<DenseRowActionMenu
      ariaLabel={t.actions.openMenu}
      leadingActions={<DenseRowActionButton icon={<RunIcon aria-hidden="true" />} label={t.tooltips.initiateScan} onClick={onInitiateScan} />}
    >
      <DropdownMenuItem onClick={onView}>
        <ViewIcon />
        {t.tooltips.targetSummary}
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem onClick={onScheduleScan}>
        <ScheduleIcon />
        {t.actions.scheduleScan}
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem onClick={onDelete} className="focus:text-destructive text-destructive">
        <DeleteIcon />
        {t.actions.delete}
      </DropdownMenuItem>
    </DenseRowActionMenu>);
});
/**
 * Create all targets table column definitions
 */
export const createAllTargetsColumns = ({ formatDate, navigate, handleDelete, handleInitiateScan, handleScheduleScan, t, }: CreateColumnsProps): ColumnDef<Target>[] => [
    {
        id: "select",
        size: allTargetsTableColumnLayout.select.size,
        minSize: allTargetsTableColumnLayout.select.minSize,
        maxSize: allTargetsTableColumnLayout.select.maxSize,
        enableResizing: false,
        header: ({ table }) => (<Checkbox checked={table.getIsAllPageRowsSelected()} indeterminate={!table.getIsAllPageRowsSelected() && table.getIsSomePageRowsSelected()} onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)} aria-label={t.actions.selectAll}/>),
        cell: ({ row }) => (<Checkbox checked={row.getIsSelected()} onCheckedChange={(value) => row.toggleSelected(!!value)} aria-label={t.actions.selectRow}/>),
        enableSorting: false,
        enableHiding: false,
    },
    {
        accessorKey: "name",
        size: allTargetsTableColumnLayout.name.size,
        minSize: allTargetsTableColumnLayout.name.minSize,
        meta: {
            title: t.columns.target,
            orderBy: "displayName",
            firstSortDirection: "asc",
            serverSortPerformance: "covered by idx_target_name_id_active for active targets",
        },
        header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.target}/>),
        cell: ({ row }) => (<TargetNameCell name={row.getValue("name") as string} targetId={row.original.id} targetType={row.original.type} t={t}/>),
    },
    {
        accessorKey: "organizations",
        size: allTargetsTableColumnLayout.organizations.size,
        minSize: allTargetsTableColumnLayout.organizations.minSize,
        maxSize: allTargetsTableColumnLayout.organizations.maxSize,
        meta: { title: t.columns.organization },
        header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.organization}/>),
        cell: ({ row }) => {
            const organizations = row.getValue("organizations") as Array<{
                id: number;
                name: string;
            }> | undefined;
            return <ExpandableBadgeList items={organizations} maxVisible={3} singleLinePreview variant="secondary"/>;
        },
        enableSorting: false,
    },
    {
        accessorKey: "createdAt",
        size: allTargetsTableColumnLayout.createdAt.size,
        minSize: allTargetsTableColumnLayout.createdAt.minSize,
        maxSize: allTargetsTableColumnLayout.createdAt.maxSize,
        meta: {
            title: t.columns.addedOn,
            orderBy: "createdAt",
            firstSortDirection: "desc",
            serverSortPerformance: "covered by idx_target_created_at_id_active for active targets",
        },
        header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.addedOn}/>),
        cell: ({ row }) => {
            const createdAt = row.getValue("createdAt") as string;
            return <TimestampCell value={formatDate(createdAt)} className="truncate"/>;
        },
    },
    {
        accessorKey: "lastScannedAt",
        size: allTargetsTableColumnLayout.lastScannedAt.size,
        minSize: allTargetsTableColumnLayout.lastScannedAt.minSize,
        maxSize: allTargetsTableColumnLayout.lastScannedAt.maxSize,
        meta: {
            title: t.columns.lastScanned,
            orderBy: "lastScannedAt",
            firstSortDirection: "desc",
            serverSortPerformance: "covered by idx_target_last_scanned_at_desc_id_active and idx_target_last_scanned_at_asc_id_active for NULLS LAST sorting",
        },
        header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.lastScanned}/>),
        cell: ({ row }) => {
            const lastScannedAt = row.original.lastScannedAt;
            return <TimestampCell value={lastScannedAt ? formatDate(lastScannedAt) : "-"} className="truncate"/>;
        },
    },
    {
        id: "actions",
        size: allTargetsTableColumnLayout.actions.size,
        minSize: allTargetsTableColumnLayout.actions.minSize,
        maxSize: allTargetsTableColumnLayout.actions.maxSize,
        enableResizing: false,
        header: () => <span className="sr-only">{t.columns.actions}</span>,
        cell: ({ row }) => (
            <div className="flex justify-end">
                <TargetRowActions
                    onView={() => navigate(`/targets/${row.original.id}/overview/`)}
                    onInitiateScan={() => handleInitiateScan(row.original)}
                    onScheduleScan={() => handleScheduleScan(row.original)}
                    onDelete={() => handleDelete(row.original)}
                    t={t}
                />
            </div>
        ),
        enableSorting: false,
        enableHiding: false,
    },
];
