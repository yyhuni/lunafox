"use client"

import type { ColumnDef } from "@tanstack/react-table"
import { useTranslations } from "next-intl"

import { semanticIcons } from "@/components/icons"
import { DataTableColumnHeader } from "@/components/shared/data-table"
import { DenseRowActionMenu } from "@/components/shared/data-table/menu-owners"
import { Badge } from "@/components/ui/badge"
import { DropdownMenuItem } from "@/components/ui/dropdown-menu"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

export type WorkflowManagementColumnRow = {
  scanWorkflowId: string
  title: string
  name: string
  description?: string
  engineCount: number
  isBuiltin: boolean
  isExecutable: boolean
}

export function createWorkflowManagementColumns<TData extends WorkflowManagementColumnRow>({
  tWorkflow,
  editLabel,
  viewLabel,
  builtinLabel,
  unavailableLabel,
  actionMenuLabel,
  onEdit,
}: {
  tWorkflow: ReturnType<typeof useTranslations>
  editLabel: string
  viewLabel: string
  builtinLabel: string
  unavailableLabel: string
  actionMenuLabel: string
  onEdit?: (workflow: TData) => void
}): ColumnDef<TData>[] {
  return [
    {
      accessorKey: "scanWorkflowId",
      size: 180,
      minSize: 160,
      meta: { title: tWorkflow("management.columns.id") },
      header: ({ column }) => <DataTableColumnHeader column={column} title={tWorkflow("management.columns.id")} />,
      cell: ({ row }) => (
        <span className={cn("truncate font-mono", textRole.tableCellSecondary)}>{row.original.scanWorkflowId}</span>
      ),
    },
    {
      accessorKey: "title",
      size: 240,
      minSize: 200,
      meta: { title: tWorkflow("management.columns.name") },
      header: ({ column }) => <DataTableColumnHeader column={column} title={tWorkflow("management.columns.name")} />,
      cell: ({ row }) => (
        <span className="flex min-w-0 items-center gap-2">
          <span className={cn("truncate", textRole.tableCellPrimary)}>{row.original.title}</span>
          {row.original.isBuiltin ? <Badge variant="secondary">{builtinLabel}</Badge> : null}
          {!row.original.isExecutable ? <Badge variant="outline">{unavailableLabel}</Badge> : null}
        </span>
      ),
    },
    {
      accessorKey: "description",
      size: 280,
      minSize: 220,
      meta: { title: tWorkflow("management.columns.description") },
      header: ({ column }) => <DataTableColumnHeader column={column} title={tWorkflow("management.columns.description")} />,
      cell: ({ row }) => (
        <div className={cn("line-clamp-2", textRole.tableCellSecondary)}>
          {row.original.description || tWorkflow("management.noDescription")}
        </div>
      ),
      enableSorting: false,
    },
    {
      accessorKey: "engineCount",
      size: 76,
      minSize: 72,
      maxSize: 88,
      meta: { title: tWorkflow("management.columns.engines") },
      header: ({ column }) => <DataTableColumnHeader column={column} title={tWorkflow("management.columns.engines")} />,
      cell: ({ row }) => <span className="tabular-nums">{row.original.engineCount || "-"}</span>,
    },
    {
      id: "actions",
      size: 64,
      minSize: 64,
      maxSize: 64,
      enableResizing: false,
      enableSorting: false,
      enableHiding: false,
      header: () => <span className="sr-only">{editLabel}</span>,
      cell: ({ row }) => (
        <DenseRowActionMenu
          ariaLabel={actionMenuLabel}
        >
          <DropdownMenuItem onClick={() => onEdit?.(row.original)}>
            <semanticIcons.action.edit />
            {row.original.isBuiltin ? viewLabel : editLabel}
          </DropdownMenuItem>
        </DenseRowActionMenu>
      ),
    },
  ]
}
