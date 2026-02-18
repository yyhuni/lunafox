"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { Badge } from "@/components/ui/badge"
import { Checkbox } from "@/components/ui/checkbox"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import type { Endpoint } from "@/types/endpoint.types"
import { ExpandableCell, ExpandableTagList } from "@/components/ui/data-table/expandable-cell"

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
		responseTime: string
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
}

function HttpStatusBadge({ statusCode }: { statusCode: number | null | undefined }) {
  if (statusCode === null || statusCode === undefined) {
    return (
      <Badge variant="outline" className="text-muted-foreground px-2 py-1 font-mono">
        -
      </Badge>
    )
  }

  const getStatusVariant = (code: number): "default" | "secondary" | "destructive" | "outline" => {
    if (code >= 200 && code < 300) {
      return "outline"
    } else if (code >= 300 && code < 400) {
      return "secondary"
    } else if (code >= 400 && code < 500) {
      return "default"
    } else if (code >= 500) {
      return "destructive"
    } else {
      return "secondary"
    }
  }

  const variant = getStatusVariant(statusCode)

  return (
    <Badge variant={variant} className="px-2 py-1 font-mono tabular-nums">
      {statusCode}
    </Badge>
  )
}

export function createEndpointColumns({
  formatDate,
  t,
}: CreateColumnsProps): ColumnDef<Endpoint>[] {
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
      meta: { title: t.columns.url },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.url} />
      ),
      size: 400,
      minSize: 200,
      maxSize: 700,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("url")} />
      ),
    },
    {
      accessorKey: "host",
      meta: { title: t.columns.host },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.host} />
      ),
      size: 200,
      minSize: 100,
      maxSize: 300,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("host")} />
      ),
    },
    {
      accessorKey: "title",
      meta: { title: t.columns.title },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.title} />
      ),
      size: 150,
      minSize: 100,
      maxSize: 300,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("title")} />
      ),
    },
    {
      accessorKey: "statusCode",
      meta: { title: t.columns.status },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.status} />
      ),
      size: 80,
      minSize: 60,
      maxSize: 100,
      cell: ({ row }) => {
        const status = row.getValue("statusCode") as number | null | undefined
        return <HttpStatusBadge statusCode={status} />
      },
    },
    {
      accessorKey: "contentLength",
      meta: { title: t.columns.contentLength },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.contentLength} />
      ),
      size: 100,
      minSize: 80,
      maxSize: 150,
      cell: ({ row }) => {
        const len = row.getValue("contentLength") as number | null | undefined
        if (len === null || len === undefined) {
          return <span className="text-muted-foreground text-sm">-</span>
        }
        return <span className="font-mono tabular-nums">{new Intl.NumberFormat().format(len)}</span>
      },
    },
    {
      accessorKey: "location",
      meta: { title: t.columns.location },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.location} />
      ),
      size: 150,
      minSize: 100,
      maxSize: 300,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("location")} />
      ),
    },
    {
      accessorKey: "webserver",
      meta: { title: t.columns.webServer },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.webServer} />
      ),
      size: 120,
      minSize: 80,
      maxSize: 200,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("webserver")} />
      ),
    },
    {
      accessorKey: "contentType",
      meta: { title: t.columns.contentType },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.contentType} />
      ),
      size: 120,
      minSize: 80,
      maxSize: 200,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("contentType")} />
      ),
    },
    {
      accessorKey: "tech",
      meta: { title: t.columns.technologies },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.technologies} />
      ),
      size: 200,
      minSize: 150,
      cell: ({ row }) => {
        const tech = (row.getValue("tech") as string[] | null | undefined) || []
        return (
          <ExpandableTagList
            items={tech}
            maxLines={2}
            variant="outline"
          />
        )
      },
    },
    {
      accessorKey: "responseBody",
      meta: { title: t.columns.responseBody },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.responseBody} />
      ),
      size: 350,
      minSize: 250,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("responseBody")} />
      ),
    },
    {
      accessorKey: "responseHeaders",
      meta: { title: t.columns.responseHeaders },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.responseHeaders} />
      ),
      size: 250,
      minSize: 150,
      maxSize: 400,
      cell: ({ row }) => {
        const headers = row.getValue("responseHeaders") as string | null | undefined
        if (!headers) return <span className="text-muted-foreground text-sm">-</span>
        return <ExpandableCell value={headers} maxLines={3} />
      },
    },
    {
      accessorKey: "vhost",
      meta: { title: t.columns.vhost },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.vhost} />
      ),
      size: 80,
      minSize: 60,
      maxSize: 100,
      cell: ({ row }) => {
        const vhost = row.getValue("vhost") as boolean | null | undefined
        if (vhost === null || vhost === undefined) return <span className="text-sm text-muted-foreground">-</span>
        return <span className="text-sm font-mono">{vhost ? "true" : "false"}</span>
      },
    },
	{
		accessorKey: "responseTime",
      meta: { title: t.columns.responseTime },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.responseTime} />
      ),
      size: 100,
      minSize: 80,
      maxSize: 150,
      cell: ({ row }) => {
        const rt = row.getValue("responseTime") as number | null | undefined
        if (rt === null || rt === undefined) {
          return <span className="text-muted-foreground text-sm">-</span>
        }
        const formatted = `${rt.toFixed(4)}s`
        return <span className="font-mono text-emerald-600 dark:text-emerald-400">{formatted}</span>
      },
    },
    {
      accessorKey: "createdAt",
      meta: { title: t.columns.createdAt },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.createdAt} />
      ),
      size: 150,
      minSize: 120,
      maxSize: 200,
      cell: ({ row }) => {
        const createdAt = row.getValue("createdAt") as string | undefined
        return <div className="text-sm">{createdAt ? formatDate(createdAt) : "-"}</div>
      },
    },
  ]
}
