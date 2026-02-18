"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import type { WebSite } from "@/types/website.types"
import { ExpandableCell, ExpandableTagList } from "@/components/ui/data-table/expandable-cell"

// Translation type definitions
export interface WebsiteTranslations {
  columns: {
    url: string
    host: string
    title: string
    status: string
    technologies: string
    contentLength: string
    location: string
    webServer: string
    contentType: string
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

interface CreateWebSiteColumnsProps {
  formatDate: (dateString: string) => string
  t: WebsiteTranslations
}

export function createWebSiteColumns({
  formatDate,
  t,
}: CreateWebSiteColumnsProps): ColumnDef<WebSite>[] {
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
      maxSize: 250,
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
      enableResizing: false,
      cell: ({ row }) => {
        const statusCode = row.getValue("statusCode") as number
        if (!statusCode) return "-"
        
        let variant: "default" | "secondary" | "destructive" | "outline" = "default"
        if (statusCode >= 200 && statusCode < 300) {
          variant = "default"
        } else if (statusCode >= 300 && statusCode < 400) {
          variant = "secondary"
        } else if (statusCode >= 400) {
          variant = "destructive"
        }
        
        return <Badge variant={variant}>{statusCode}</Badge>
      },
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
        const tech = row.getValue("tech") as string[]
        return <ExpandableTagList items={tech} maxLines={2} variant="outline" />
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
        const contentLength = row.getValue("contentLength") as number
        if (!contentLength) return "-"
        return contentLength.toString()
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
        const headers = row.getValue("responseHeaders") as string | null
        if (!headers) return "-"
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
      enableResizing: false,
      cell: ({ row }) => {
        const vhost = row.getValue("vhost") as boolean | null
        if (vhost === null) return "-"
        return (
          <Badge variant={vhost ? "default" : "secondary"} className="text-xs">
            {vhost ? "true" : "false"}
          </Badge>
        )
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
      enableResizing: false,
      cell: ({ row }) => {
        const createdAt = row.getValue("createdAt") as string
        return <div className="text-sm">{createdAt ? formatDate(createdAt) : "-"}</div>
      },
    },
  ]
}
