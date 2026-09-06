"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { useLocale, useTranslations } from "next-intl"
import { Checkbox } from "@/components/ui/checkbox"
import { DataTableColumnHeader } from "@/components/shared/data-table/column-header"
import { ExpandableCell } from "@/components/shared/data-table/expandable-cell"
import { HttpStatusBadge } from "@/components/shared/status/http-status-badge"
import { SingleBadgeCell } from "@/components/shared/data-table/single-badge-cell"
import { TimestampCell } from "@/components/shared/data-table/timestamp-cell"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { Directory } from "@/types/directory.types"
import { getDateLocale } from "@/lib/date-utils"

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
  includeSelection?: boolean
}

export function useDirectoryTableColumns({ includeSelection = true }: { includeSelection?: boolean } = {}) {
  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const locale = useLocale()

  const translations = React.useMemo<DirectoryTranslations>(
    () => ({
      columns: {
        url: tColumns("common.url"),
        status: tColumns("common.status"),
        length: tColumns("directory.length"),
        contentType: tColumns("endpoint.contentType"),
        createdAt: tColumns("common.createdAt"),
      },
      actions: {
        selectAll: tCommon("actions.selectAll"),
        selectRow: tCommon("actions.selectRow"),
      },
    }),
    [tColumns, tCommon]
  )

  const formatDate = React.useCallback(
    (dateString: string) => new Date(dateString).toLocaleString(getDateLocale(locale), {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
    }),
    [locale]
  )

  const columns = React.useMemo(
    () => createDirectoryColumns({ formatDate, t: translations, includeSelection }),
    [formatDate, includeSelection, translations]
  )

  return { columns, formatDate }
}

/**
 * Create directory table column definitions
 */
export function createDirectoryColumns({
  formatDate,
  t,
  includeSelection = true,
}: CreateColumnsProps): ColumnDef<Directory>[] {
  const selectionColumn: ColumnDef<Directory> = {
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
    }

  const dataColumns: ColumnDef<Directory>[] = [
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
        <ExpandableCell value={row.getValue("url")} textRoleName="tableCellPrimary" />
      ),
    },
    {
      accessorKey: "status",
      size: 80,
      minSize: 60,
      maxSize: 120,
      enableResizing: false,
      meta: {
        title: t.columns.status,
        orderBy: "status",
        firstSortDirection: "asc",
        serverSortPerformance: "indexed",
      },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.status} />
      ),
      cell: ({ row }) => <HttpStatusBadge statusCode={row.getValue("status")} />,
    },
    {
      accessorKey: "contentLength",
      size: 100,
      minSize: 80,
      maxSize: 150,
      meta: {
        title: t.columns.length,
        orderBy: "contentLength",
        firstSortDirection: "desc",
        serverSortPerformance: "indexed",
      },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.length} />
      ),
      cell: ({ row }) => {
        const length = row.getValue("contentLength") as string | null
        return (
          <span className={cn(textRole.tableCellSecondary, "font-mono tabular-nums")}>
            {length ?? "-"}
          </span>
        )
      },
    },
    {
      accessorKey: "contentType",
      size: 120,
      minSize: 80,
      maxSize: 200,
      enableResizing: false,
      meta: {
        title: t.columns.contentType,
        singleBadge: true,
      },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.contentType} />
      ),
      cell: ({ row }) => <SingleBadgeCell value={row.getValue("contentType")} />,
    },
    {
      accessorKey: "createdAt",
      size: 176,
      minSize: 176,
      maxSize: 220,
      enableResizing: false,
      meta: {
        title: t.columns.createdAt,
        orderBy: "createdAt",
        firstSortDirection: "desc",
        serverSortPerformance: "indexed",
      },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.createdAt} />
      ),
      cell: ({ row }) => {
        const date = row.getValue("createdAt") as string
        return <TimestampCell value={formatDate(date)} />
      },
    },
  ]

  return includeSelection ? [selectionColumn, ...dataColumns] : dataColumns
}
