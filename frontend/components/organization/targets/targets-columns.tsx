"use client";
import React from "react";
import { ColumnDef } from "@tanstack/react-table";
import Link from "next/link";
import { Checkbox } from "@/components/ui/checkbox";
import { DropdownMenuItem, } from "@/components/ui/dropdown-menu";
import { Eye, Trash2, semanticIcons } from "@/components/icons";
import { DataTableColumnHeader } from "@/components/shared/data-table/column-header";
import { DenseRowActionMenu } from "@/components/shared/data-table/menu-owners";
import { QuietCopyButton } from "@/components/shared/data-table/row-actions";
import { SingleBadgeCell } from "@/components/shared/data-table/single-badge-cell";
import { TimestampCell } from "@/components/shared/data-table/timestamp-cell";
import { textRole } from "@/lib/typography";
import { cn } from "@/lib/utils";
import type { Target, TargetType } from "@/types/target.types";
const TARGET_NAME_COPY_TOAST_ID = "target-name-copy";
// Translation type definition
export interface OrgTargetsTranslations {
    columns: {
        targetName: string;
        type: string;
        addedOn: string;
        lastScanned: string;
    };
    actions: {
        selectAll: string;
        selectRow: string;
    };
    tooltips: {
        viewDetails: string;
        unlinkTarget: string;
        clickToCopy: string;
        copied: string;
    };
    types: {
        domain: string;
        ip: string;
        cidr: string;
    };
}
interface CreateColumnsProps {
    formatDate: (dateString: string) => string;
    navigate: (path: string) => void;
    handleDelete: (target: Target) => void;
    t: OrgTargetsTranslations;
}
function getTargetTypeInfo(type: string | null | undefined, t: OrgTargetsTranslations) {
    if (!type) {
        return null;
    }
    const typeMap: Record<string, {
        label: string;
        variant: "default" | "secondary" | "outline";
    }> = {
        domain: { label: t.types.domain, variant: "default" },
        ip: { label: t.types.ip, variant: "secondary" },
        cidr: { label: t.types.cidr, variant: "outline" },
    };
    return typeMap[type] || { label: type, variant: "secondary" as const };
}
/**
 * target row operation component
 */
function TargetRowActions({ onView, onDelete, t, }: {
    onView: () => void;
    onDelete: () => void;
    t: OrgTargetsTranslations;
}) {
    return (<DenseRowActionMenu ariaLabel={`${t.tooltips.viewDetails} / ${t.tooltips.unlinkTarget}`}>
      <DropdownMenuItem onClick={onView}>
          <Eye />
          {t.tooltips.viewDetails}
        </DropdownMenuItem>
      <DropdownMenuItem onClick={onDelete} variant="destructive">
        <Trash2 />
        {t.tooltips.unlinkTarget}
      </DropdownMenuItem>
    </DenseRowActionMenu>);
}
/**
 * target name cell component
 */
function TargetNameCell({ name, targetId, targetType, t, }: {
    name: string;
    targetId: number;
    targetType: TargetType;
    t: OrgTargetsTranslations;
}) {
    const TargetTypeIcon = semanticIcons.concept[targetType];
    const targetTypeLabel = getTargetTypeInfo(targetType, t)?.label ?? targetType;

    return (<div className="group flex min-w-0 items-center gap-2">
      <TargetTypeIcon role="img" aria-label={targetTypeLabel} title={targetTypeLabel} className="size-4 shrink-0 text-muted-foreground transition-colors group-hover:text-foreground group-data-[state=selected]:text-foreground" />
      <div className="flex min-w-0 max-w-full flex-1 items-center gap-1">
        <Link href={`/targets/${targetId}/overview/`} className={cn("block min-w-0 truncate text-left transition-colors underline-offset-2 hover:text-primary hover:underline", textRole.tableCellPrimary)}>
          {name}
        </Link>
        <QuietCopyButton value={name} copyLabel={t.tooltips.clickToCopy} copiedLabel={t.tooltips.copied} toastId={TARGET_NAME_COPY_TOAST_ID}/>
      </div>
    </div>);
}
/**
 * Create target table column definitions
 */
export const createTargetColumns = ({ formatDate, navigate, handleDelete, t, }: CreateColumnsProps): ColumnDef<Target>[] => {
    return [
        {
            id: "select",
            size: 40,
            minSize: 40,
            maxSize: 40,
            enableResizing: false,
            header: ({ table }) => (<Checkbox checked={table.getIsAllPageRowsSelected()} indeterminate={!table.getIsAllPageRowsSelected() && table.getIsSomePageRowsSelected()} onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)} aria-label={t.actions.selectAll}/>),
            cell: ({ row }) => (<Checkbox checked={row.getIsSelected()} onCheckedChange={(value) => row.toggleSelected(!!value)} aria-label={t.actions.selectRow}/>),
            enableSorting: false,
            enableHiding: false,
        },
        {
            accessorKey: "name",
            size: 350,
            minSize: 250,
            meta: { title: t.columns.targetName },
            header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.targetName}/>),
            cell: ({ row }) => (<TargetNameCell name={row.getValue("name") as string} targetId={row.original.id} targetType={row.original.type} t={t}/>),
        },
        {
            accessorKey: "type",
            size: 100,
            minSize: 80,
            maxSize: 140,
            meta: {
                title: t.columns.type,
                singleBadge: true,
                singleBadgeValue: (row) => getTargetTypeInfo(row.type, t)?.label ?? null,
            },
            header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.type}/>),
            cell: ({ row }) => {
                const type = row.getValue("type") as string | null;
                const typeInfo = getTargetTypeInfo(type, t);
                return <SingleBadgeCell value={typeInfo?.label} variant={typeInfo?.variant}/>;
            },
        },
        {
            accessorKey: "createdAt",
            size: 176,
            minSize: 176,
            maxSize: 220,
            meta: { title: t.columns.addedOn },
            header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.addedOn}/>),
            cell: ({ row }) => {
                const createdAt = row.getValue("createdAt") as string;
                return <TimestampCell value={formatDate(createdAt)} className="truncate"/>;
            },
        },
        {
            accessorKey: "lastScannedAt",
            size: 176,
            minSize: 176,
            maxSize: 220,
            meta: { title: t.columns.lastScanned },
            header: ({ column }) => (<DataTableColumnHeader column={column} title={t.columns.lastScanned}/>),
            cell: ({ row }) => {
                const lastScannedAt = row.original.lastScannedAt;
                return <TimestampCell value={lastScannedAt ? formatDate(lastScannedAt) : "-"} className="truncate"/>;
            },
        },
        {
            id: "actions",
            size: 56,
            minSize: 56,
            maxSize: 56,
            enableResizing: false,
            meta: { stickyRight: true },
            cell: ({ row }) => (<TargetRowActions onView={() => navigate(`/targets/${row.original.id}/overview/`)} onDelete={() => handleDelete(row.original)} t={t}/>),
            enableSorting: false,
            enableHiding: false,
        },
    ];
};
