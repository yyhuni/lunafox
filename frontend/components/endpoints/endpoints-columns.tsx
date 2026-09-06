"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { Checkbox } from "@/components/ui/checkbox"
import { DataTableColumnHeader } from "@/components/shared/data-table/column-header"
import { HttpStatusBadge } from "@/components/shared/status/http-status-badge"
import { MonoValueCell } from "@/components/shared/data-table/mono-value-cell"
import { TimestampCell } from "@/components/shared/data-table/timestamp-cell"
import type { Endpoint } from "@/types/endpoint.types"
import { ExpandableCell, ExpandableTagList } from "@/components/shared/data-table/expandable-cell"
import { textRole } from "@/lib/typography"

// Translation type definitions
export interface EndpointTranslations {
	columns: {
		url: string
		host: string
		title: string
    status: string
    contentLength: string
    location: string
    webServer: string
    contentType: string
		technologies: string
		responseBody: string
		vhost: string
		responseHeaders: string
		createdAt: string
	}
  actions: {
    selectAll: string
    selectRow: string
  }
}

interface CreateColumnsProps {
  formatDate: (dateString: string) => string
  t: EndpointTranslations
  includeSelection?: boolean
}

export function createEndpointColumns({
  formatDate,
  t,
  includeSelection = true,
}: CreateColumnsProps): ColumnDef<Endpoint>[] {
  const selectionColumn: ColumnDef<Endpoint> = {
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

  const dataColumns: ColumnDef<Endpoint>[] = [
    {
      accessorKey: "url",
      enableSorting: false,
      meta: { title: t.columns.url, widthPolicy: { mode: "flex", flex: 2, fill: true } },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.url} />
      ),
      size: 220,
      minSize: 152,
      maxSize: 320,
      cell: ({ row }) => (
        <div className="flex min-w-0 items-center gap-2">
          <HttpStatusBadge statusCode={row.original.statusCode} />
          <div className="min-w-0 flex-1">
            <ExpandableCell value={row.getValue("url")} textRoleName="tableCellPrimary" />
          </div>
        </div>
      ),
    },
    {
      accessorKey: "host",
      enableSorting: false,
      meta: { title: t.columns.host },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.host} />
      ),
      size: 104,
      minSize: 88,
      maxSize: 150,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("host")} textRoleName="tableCellPrimary" />
      ),
    },
    {
      accessorKey: "title",
      enableSorting: false,
      meta: { title: t.columns.title, widthPolicy: { mode: "flex", flex: 1.25 } },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.title} />
      ),
      size: 144,
      minSize: 112,
      maxSize: 260,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("title")} textRoleName="tableCellPrimary" />
      ),
    },
    {
      accessorKey: "contentLength",
      meta: {
        title: t.columns.contentLength,
        orderBy: "contentLength",
        firstSortDirection: "desc",
        serverSortPerformance: "indexed",
      },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.contentLength} />
      ),
      size: 76,
      minSize: 72,
      maxSize: 96,
      cell: ({ row }) => {
        const len = row.getValue("contentLength") as number | null | undefined
        if (len === null || len === undefined) {
          return <MonoValueCell value={null} />
        }
        return (
          <MonoValueCell
            value={new Intl.NumberFormat().format(len)}
            contentClassName="tabular-nums"
          />
        )
      },
    },
    {
      accessorKey: "location",
      enableSorting: false,
      meta: { title: t.columns.location },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.location} />
      ),
      size: 96,
      minSize: 88,
      maxSize: 180,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("location")} textRoleName="tableCellSecondary" />
      ),
    },
    {
      accessorKey: "webserver",
      enableSorting: false,
      meta: { title: t.columns.webServer },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.webServer} />
      ),
      size: 100,
      minSize: 88,
      maxSize: 160,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("webserver")} textRoleName="tableCellSecondary" />
      ),
    },
    {
      accessorKey: "contentType",
      enableSorting: false,
      meta: { title: t.columns.contentType },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.contentType} />
      ),
      size: 100,
      minSize: 96,
      maxSize: 180,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("contentType")} textRoleName="tableCellSecondary" />
      ),
    },
    {
      accessorKey: "tech",
      enableSorting: false,
      meta: { title: t.columns.technologies, widthPolicy: { mode: "flex", flex: 0.75 } },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.technologies} />
      ),
      size: 120,
      minSize: 96,
      maxSize: 240,
      cell: ({ row }) => {
        const tech = (row.getValue("tech") as string[] | null | undefined) || []
        return (
          <ExpandableTagList
            items={tech}
            maxVisible={3}
            singleLinePreview
            maxLines={2}
            variant="outline"
          />
        )
      },
    },
    {
      accessorKey: "responseBody",
      enableSorting: false,
      meta: { title: t.columns.responseBody },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.responseBody} />
      ),
      size: 90,
      minSize: 88,
      maxSize: 220,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("responseBody")} textRoleName="tableCellSecondary" />
      ),
    },
    {
      accessorKey: "responseHeaders",
      enableSorting: false,
      meta: { title: t.columns.responseHeaders },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.responseHeaders} />
      ),
      size: 90,
      minSize: 88,
      maxSize: 220,
      cell: ({ row }) => {
        const headers = row.getValue("responseHeaders") as string | null | undefined
        if (!headers) return <span className={textRole.tableCellSecondary}>-</span>
        return <ExpandableCell value={headers} maxLines={3} textRoleName="tableCellSecondary" />
      },
    },
    {
      accessorKey: "vhost",
      enableSorting: false,
      meta: { title: t.columns.vhost },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.vhost} />
      ),
      size: 66,
      minSize: 60,
      maxSize: 88,
      cell: ({ row }) => {
        const vhost = row.getValue("vhost") as boolean | null | undefined
        if (vhost === null || vhost === undefined) return <MonoValueCell value={null} />
        return <MonoValueCell value={vhost ? "true" : "false"} />
      },
    },
    {
      accessorKey: "createdAt",
      meta: {
        title: t.columns.createdAt,
        orderBy: "createdAt",
        firstSortDirection: "desc",
        serverSortPerformance: "indexed",
      },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.createdAt} />
      ),
      size: 176,
      minSize: 176,
      maxSize: 220,
      enableResizing: false,
      cell: ({ row }) => {
        const createdAt = row.getValue("createdAt") as string | undefined
        return <TimestampCell value={createdAt ? formatDate(createdAt) : "-"} />
      },
    },
  ]

  return includeSelection ? [selectionColumn, ...dataColumns] : dataColumns
}
