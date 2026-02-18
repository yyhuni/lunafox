"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import { ExpandableCell } from "@/components/ui/data-table/expandable-cell"
import type { Directory } from "@/types/directory.types"

// Translation type definitions
export interface DirectoryTranslations {
  columns: {
    url: string
    status: string
    length: string
    contentType: string
    createdAt: string
  }
  actions: {
    selectAll: string
    selectRow: string
  }
}

interface CreateColumnsProps {
  formatDate: (date: string) => string
  t: DirectoryTranslations
}

/**
 * HTTP status code badge component
 */
function StatusBadge({ status }: { status: number | null }) {
  if (!status) return <span className="text-muted-foreground">-</span>

  let className = ""

  if (status >= 200 && status < 300) {
    className = "bg-green-500/10 text-green-700 dark:text-green-400 hover:bg-green-500/20"
  } else if (status >= 300 && status < 400) {
    className = "bg-blue-500/10 text-blue-700 dark:text-blue-400 hover:bg-blue-500/20"
  } else if (status >= 400 && status < 500) {
    className = "bg-yellow-500/10 text-yellow-700 dark:text-yellow-400 hover:bg-yellow-500/20"
  } else if (status >= 500) {
    className = "bg-red-500/10 text-red-700 dark:text-red-400 hover:bg-red-500/20"
  }

  return (
    <Badge variant="default" className={className}>
      {status}
    </Badge>
  )
}

/**
 * Create directory table column definitions
 */
export function createDirectoryColumns({
  formatDate,
  t,
}: CreateColumnsProps): ColumnDef<Directory>[] {
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
      accessorKey: "url",
      size: 400,
      minSize: 200,
      maxSize: 800,
      meta: { title: t.columns.url },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.url} />
      ),
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("url")} />
      ),
    },
    {
      accessorKey: "status",
      size: 80,
      minSize: 60,
      maxSize: 120,
      enableResizing: false,
      meta: { title: t.columns.status },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.status} />
      ),
      cell: ({ row }) => <StatusBadge status={row.getValue("status")} />,
    },
    {
      accessorKey: "contentLength",
      size: 100,
      minSize: 80,
      maxSize: 150,
      meta: { title: t.columns.length },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.length} />
      ),
      cell: ({ row }) => {
        const length = row.getValue("contentLength") as number | null
        return <span>{length !== null ? length.toLocaleString() : "-"}</span>
      },
    },
    {
      accessorKey: "contentType",
      size: 120,
      minSize: 80,
      maxSize: 200,
      enableResizing: false,
      meta: { title: t.columns.contentType },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.contentType} />
      ),
      cell: ({ row }) => {
        const contentType = row.getValue("contentType") as string
        return contentType ? (
          <Badge variant="outline">{contentType}</Badge>
        ) : (
          <span className="text-muted-foreground">-</span>
        )
      },
    },
    {
      accessorKey: "createdAt",
      size: 150,
      minSize: 120,
      maxSize: 200,
      enableResizing: false,
      meta: { title: t.columns.createdAt },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.createdAt} />
      ),
      cell: ({ row }) => {
        const date = row.getValue("createdAt") as string
        return <span className="text-muted-foreground">{formatDate(date)}</span>
      },
    },
  ]
}
