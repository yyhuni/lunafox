"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { useLocale, useTranslations } from "next-intl"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import { DataTableColumnHeader } from "@/components/shared/data-table/column-header"
import { HttpStatusBadge } from "@/components/shared/status/http-status-badge"
import type { WebSite } from "@/types/website.types"
import { ExpandableCell, ExpandableTagList } from "@/components/shared/data-table/expandable-cell"
import { TimestampCell } from "@/components/shared/data-table/timestamp-cell"
import { textRole } from "@/lib/typography"
import { getDateLocale } from "@/lib/date-utils"

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

export function useWebSiteTableColumns() {
  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const locale = useLocale()

  const translations = React.useMemo<WebsiteTranslations>(
    () => ({
      columns: {
        url: tColumns("common.url"),
        host: tColumns("website.host"),
        title: tColumns("endpoint.title"),
        status: tColumns("website.statusCode"),
        technologies: tColumns("endpoint.technologies"),
        contentLength: tColumns("endpoint.contentLength"),
        location: tColumns("endpoint.location"),
        webServer: tColumns("endpoint.webServer"),
        contentType: tColumns("endpoint.contentType"),
        responseBody: tColumns("endpoint.responseBody"),
        vhost: tColumns("endpoint.vhost"),
        responseHeaders: tColumns("website.responseHeaders"),
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
    () => createWebSiteColumns({ formatDate, t: translations }),
    [formatDate, translations]
  )

  return { columns, formatDate }
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
        const tech = row.getValue("tech") as string[]
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
        const contentLength = row.getValue("contentLength") as number
        if (!contentLength) return <span className={textRole.tableCellSecondary}>-</span>
        return (
          <span className={`${textRole.tableCellSecondary} font-mono tabular-nums`}>
            {contentLength.toString()}
          </span>
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
      size: 104,
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
      size: 108,
      minSize: 96,
      maxSize: 180,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("contentType")} textRoleName="tableCellSecondary" />
      ),
    },
    {
      accessorKey: "responseBody",
      enableSorting: false,
      meta: { title: t.columns.responseBody },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.responseBody} />
      ),
      size: 112,
      minSize: 96,
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
      size: 112,
      minSize: 96,
      maxSize: 220,
      cell: ({ row }) => {
        const headers = row.getValue("responseHeaders") as string | null
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
      enableResizing: false,
      cell: ({ row }) => {
        const vhost = row.getValue("vhost") as boolean | null
        if (vhost === null) return <span className={textRole.tableCellSecondary}>-</span>
        return (
          <Badge variant={vhost ? "default" : "secondary"} className="text-xs">
            {vhost ? "true" : "false"}
          </Badge>
        )
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
        const createdAt = row.getValue("createdAt") as string
        return <TimestampCell value={createdAt ? formatDate(createdAt) : "-"} />
      },
    },
  ]
}
