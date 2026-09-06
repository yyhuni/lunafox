"use client"

import { useMemo, useCallback, useState } from "react"
import { useFormatter, useTranslations } from "next-intl"
import type { ColumnDef, VisibilityState } from "@tanstack/react-table"
import { DataTableColumnHeader } from "@/components/shared/data-table/column-header"
import { HttpStatusBadge } from "@/components/shared/status/http-status-badge"
import { UnifiedDataTable } from "@/components/shared/data-table/unified-data-table"
import { ExpandableCell, ExpandableTagList } from "@/components/shared/data-table/expandable-cell"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { EndpointSearchResult } from "@/types/search.types"

interface SearchResultsTableProps {
  results: EndpointSearchResult[]
}

const DEFAULT_SEARCH_COLUMN_VISIBILITY: VisibilityState = {
  responseBody: false,
  responseHeaders: false,
}

export function SearchResultsTable({ results }: SearchResultsTableProps) {
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>(
    DEFAULT_SEARCH_COLUMN_VISIBILITY
  )
  const format = useFormatter()
  const t = useTranslations("search.table")

  const formatDate = useCallback((dateString: string) => {
    return format.dateTime(new Date(dateString), {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    })
  }, [format])

  // Basic column definitions (common to Website and Endpoint)
  const baseColumns: ColumnDef<EndpointSearchResult, unknown>[] = useMemo(() => [
    {
      id: "url",
      accessorKey: "url",
      meta: { title: t("url") },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("url")} />
      ),
      size: 350,
      minSize: 200,
      maxSize: 600,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("url")} textRoleName="tableCellPrimary" />
      ),
    },
    {
      id: "host",
      accessorKey: "host",
      meta: { title: t("host") },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("host")} />
      ),
      size: 180,
      minSize: 100,
      maxSize: 250,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("host")} textRoleName="tableCellPrimary" />
      ),
    },
    {
      id: "title",
      accessorKey: "title",
      meta: { title: t("title") },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("title")} />
      ),
      size: 150,
      minSize: 100,
      maxSize: 300,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("title")} textRoleName="tableCellPrimary" />
      ),
    },
    {
      id: "statusCode",
      accessorKey: "statusCode",
      meta: { title: t("status") },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("status")} />
      ),
      size: 80,
      minSize: 60,
      maxSize: 100,
      cell: ({ row }) => {
        const statusCode = row.getValue("statusCode") as number | null
        return <HttpStatusBadge statusCode={statusCode} />
      },
    },
    {
      id: "tech",
      accessorKey: "tech",
      meta: { title: t("technologies") },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("technologies")} />
      ),
      size: 180,
      minSize: 120,
      maxSize: 260,
      cell: ({ row }) => {
        const tech = row.getValue("tech") as string[] | null
        if (!tech || tech.length === 0) return <span className={textRole.tableCellSecondary}>-</span>
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
      id: "contentLength",
      accessorKey: "contentLength",
      meta: { title: t("contentLength") },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("contentLength")} />
      ),
      size: 100,
      minSize: 80,
      maxSize: 150,
      cell: ({ row }) => {
        const len = row.getValue("contentLength") as number | null
        if (len === null || len === undefined) return <span className={textRole.tableCellSecondary}>-</span>
        return <span className={cn(textRole.tableCellSecondary, "font-mono tabular-nums")}>{new Intl.NumberFormat().format(len)}</span>
      },
    },
    {
      id: "location",
      accessorKey: "location",
      meta: { title: t("location") },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("location")} />
      ),
      size: 150,
      minSize: 100,
      maxSize: 300,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("location")} textRoleName="tableCellSecondary" />
      ),
    },
    {
      id: "webserver",
      accessorKey: "webserver",
      meta: { title: t("webserver") },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("webserver")} />
      ),
      size: 120,
      minSize: 80,
      maxSize: 200,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("webserver")} textRoleName="tableCellSecondary" />
      ),
    },
    {
      id: "contentType",
      accessorKey: "contentType",
      meta: { title: t("contentType") },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("contentType")} />
      ),
      size: 120,
      minSize: 80,
      maxSize: 200,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("contentType")} textRoleName="tableCellSecondary" />
      ),
    },
    {
      id: "responseBody",
      accessorKey: "responseBody",
      meta: { title: t("responseBody") },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("responseBody")} />
      ),
      size: 300,
      minSize: 200,
      maxSize: 420,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("responseBody")} maxLines={3} textRoleName="tableCellSecondary" />
      ),
    },
    {
      id: "responseHeaders",
      accessorKey: "responseHeaders",
      meta: { title: t("responseHeaders") },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("responseHeaders")} />
      ),
      size: 250,
      minSize: 150,
      maxSize: 400,
      cell: ({ row }) => {
        return <ExpandableCell value={row.getValue("responseHeaders")} maxLines={3} textRoleName="tableCellSecondary" />
      },
    },
    {
      id: "vhost",
      accessorKey: "vhost",
      meta: { title: t("vhost") },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("vhost")} />
      ),
      size: 80,
      minSize: 60,
      maxSize: 100,
      cell: ({ row }) => {
        const vhost = row.getValue("vhost") as boolean | null
        if (vhost === null || vhost === undefined) return <span className={textRole.tableCellSecondary}>-</span>
        return <span className={cn(textRole.tableCellSecondary, "font-mono")}>{vhost ? "true" : "false"}</span>
      },
    },
    {
      id: "createdAt",
      accessorKey: "createdAt",
      meta: { title: t("createdAt") },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("createdAt")} />
      ),
      size: 150,
      minSize: 120,
      maxSize: 200,
      cell: ({ row }) => {
        const createdAt = row.getValue("createdAt") as string | null
        if (!createdAt) return <span className={textRole.tableCellSecondary}>-</span>
        return <span className={textRole.tableCellSecondary}>{formatDate(createdAt)}</span>
      },
    },
  ], [formatDate, t])

	const columns = useMemo(() => baseColumns, [baseColumns])

  return (
    <UnifiedDataTable
      columns={columns}
      data={results}
      getRowId={(row) => String(row.id)}
      state={{ columnVisibility, onColumnVisibilityChange: setColumnVisibility }}
      ui={{
        showColumnVisibility: true,
        hidePagination: true,
      }}
      behavior={{
        enableRowSelection: false,
        columnLayout: "fixed",
        expandColumnIds: ["url", "title"],
      }}
    />
  )
}
