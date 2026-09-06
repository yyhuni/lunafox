"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { Command } from "@/types/command.types"
import { Checkbox } from "@/components/ui/checkbox"
import {
  DropdownMenuItem,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu"
import { Eye, Trash2, Copy } from "@/components/icons"
import { DataTableColumnHeader } from "@/components/shared/data-table/column-header"
import { ExpandableCell } from "@/components/shared/data-table/expandable-cell"
import { DenseRowActionMenu } from "@/components/shared/data-table/menu-owners"
import { SingleBadgeCell } from "@/components/shared/data-table/single-badge-cell"
import { TimestampCell } from "@/components/shared/data-table/timestamp-cell"
import { copyTextToClipboard } from "@/components/shared/feedback/copy-button"
import { toastFeedback } from "@/lib/toast-helpers"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

// Translation type definitions
export interface CommandTranslations {
  columns: {
    name: string
    tool: string
    commandTemplate: string
    description: string
    updatedAt: string
  }
  actions: {
    selectAll: string
    selectRow: string
    openMenu: string
    copyTemplate: string
    viewDetails: string
    delete: string
  }
  messages: {
    copied: string
    copyFailed: string
  }
}

interface CreateColumnsProps {
  formatDate: (date: string) => string
  t: CommandTranslations
}

/**
 * Create command table column definitions
 */
export function createCommandColumns({
  formatDate,
  t,
}: CreateColumnsProps): ColumnDef<Command>[] {
  return [
    {
      id: "select",
      size: 40,
      minSize: 40,
      maxSize: 40,
      enableResizing: false,
      header: ({ table }) => (
        <Checkbox
          checked={table.getIsAllPageRowsSelected()}
          indeterminate={
            !table.getIsAllPageRowsSelected() && table.getIsSomePageRowsSelected()
          }
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label={t.actions.selectAll}
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label={t.actions.selectRow}
        />
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: "displayName",
      size: 200,
      minSize: 150,
      meta: { title: t.columns.name },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.name} />
      ),
      cell: ({ row }) => {
        const displayName = row.getValue("displayName") as string
        const name = row.original.name
        return (
          <div className="flex flex-1 flex-col min-w-0">
            <span className={cn("break-all whitespace-normal", textRole.tableCellPrimary)}>
              {displayName || name}
            </span>
            {displayName && name && displayName !== name && (
              <span className={cn("break-all font-mono whitespace-normal", textRole.tableCellSecondary)}>
                {name}
              </span>
            )}
          </div>
        )
      },
    },
    {
      accessorKey: "tool",
      size: 120,
      minSize: 80,
      meta: {
        title: t.columns.tool,
        singleBadge: true,
        singleBadgeValue: (row) => row.tool?.name ?? null,
      },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.tool} />
      ),
      cell: ({ row }) => <SingleBadgeCell value={row.original.tool?.name} />,
    },
    {
      accessorKey: "commandTemplate",
      size: 350,
      minSize: 250,
      meta: { title: t.columns.commandTemplate },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.commandTemplate} />
      ),
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("commandTemplate")} variant="mono" textRoleName="tableCellSecondary" />
      ),
    },
    {
      accessorKey: "description",
      size: 250,
      minSize: 150,
      meta: { title: t.columns.description },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.description} />
      ),
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("description")} variant="muted" textRoleName="tableCellSecondary" />
      ),
    },
    {
      accessorKey: "updatedAt",
      size: 176,
      minSize: 176,
      maxSize: 220,
      enableResizing: false,
      meta: { title: t.columns.updatedAt },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.updatedAt} />
      ),
      cell: ({ row }) => <TimestampCell value={formatDate(row.getValue("updatedAt"))} />,
    },
    {
      id: "actions",
      size: 60,
      minSize: 60,
      maxSize: 60,
      enableResizing: false,
      cell: ({ row }) => {
        const command = row.original

        return (
          <DenseRowActionMenu ariaLabel={t.actions.openMenu}>
            <DropdownMenuItem
              onClick={async () => {
                const success = await copyTextToClipboard(command.commandTemplate)
                if (success) {
                  toastFeedback.success(t.messages.copied)
                } else {
                  toastFeedback.error(t.messages.copyFailed)
                }
              }}
            >
              <Copy />
              {t.actions.copyTemplate}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem>
              <Eye />
              {t.actions.viewDetails}
            </DropdownMenuItem>
            <DropdownMenuItem className="focus:text-destructive text-destructive">
              <Trash2 />
              {t.actions.delete}
            </DropdownMenuItem>
          </DenseRowActionMenu>
        )
      },
      enableSorting: false,
      enableHiding: false,
    },
  ]
}
