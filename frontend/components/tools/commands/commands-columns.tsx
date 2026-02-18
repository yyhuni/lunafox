"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { Command } from "@/types/command.types"
import { Checkbox } from "@/components/ui/checkbox"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { MoreHorizontal, Eye, Trash2, Copy } from "@/components/icons"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import { ExpandableCell } from "@/components/ui/data-table/expandable-cell"
import { Badge } from "@/components/ui/badge"
import { toast } from "sonner"

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
          checked={
            table.getIsAllPageRowsSelected() ||
            (table.getIsSomePageRowsSelected() && "indeterminate")
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
          <div className="flex flex-col flex-1 min-w-0">
            <span className="text-sm font-medium break-all leading-relaxed whitespace-normal">
              {displayName || name}
            </span>
            {displayName && name && displayName !== name && (
              <span className="text-xs text-muted-foreground font-mono break-all leading-relaxed whitespace-normal">
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
      meta: { title: t.columns.tool },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.tool} />
      ),
      cell: ({ row }) => {
        const tool = row.original.tool
        return (
          <div className="flex items-center gap-2">
            {tool ? (
              <Badge variant="outline">{tool.name}</Badge>
            ) : (
              <span className="text-muted-foreground text-sm">-</span>
            )}
          </div>
        )
      },
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
        <ExpandableCell value={row.getValue("commandTemplate")} variant="mono" />
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
        <ExpandableCell value={row.getValue("description")} variant="muted" />
      ),
    },
    {
      accessorKey: "updatedAt",
      size: 150,
      minSize: 120,
      meta: { title: t.columns.updatedAt },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.updatedAt} />
      ),
      cell: ({ row }) => (
        <div className="text-sm text-muted-foreground">
          {formatDate(row.getValue("updatedAt"))}
        </div>
      ),
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
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                className="flex h-8 w-8 p-0 data-[state=open]:bg-muted"
              >
                <MoreHorizontal />
                <span className="sr-only">{t.actions.openMenu}</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                onClick={async () => {
                  try {
                    await navigator.clipboard.writeText(command.commandTemplate)
                    toast.success(t.messages.copied)
                  } catch {
                    toast.error(t.messages.copyFailed)
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
              <DropdownMenuItem className="text-destructive focus:text-destructive">
                <Trash2 />
                {t.actions.delete}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )
      },
      enableSorting: false,
      enableHiding: false,
    },
  ]
}
