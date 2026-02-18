"use client"

import { useMemo, useCallback } from "react"
import { useFormatter } from "next-intl"
import type { ColumnDef } from "@tanstack/react-table"
import { Badge } from "@/components/ui/badge"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"
import { ExpandableCell, ExpandableTagList } from "@/components/ui/data-table/expandable-cell"
import type { SearchResult, AssetType, Vulnerability } from "@/types/search.types"

interface SearchResultsTableProps {
  results: SearchResult[]
  assetType: AssetType
  onViewVulnerability?: (vuln: Vulnerability) => void
}

export function SearchResultsTable({ results }: SearchResultsTableProps) {
  const format = useFormatter()

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
  const baseColumns: ColumnDef<SearchResult, unknown>[] = useMemo(() => [
    {
      id: "url",
      accessorKey: "url",
      meta: { title: "URL" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="URL" />
      ),
      size: 350,
      minSize: 200,
      maxSize: 600,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("url")} />
      ),
    },
    {
      id: "host",
      accessorKey: "host",
      meta: { title: "Host" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Host" />
      ),
      size: 180,
      minSize: 100,
      maxSize: 250,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("host")} />
      ),
    },
    {
      id: "title",
      accessorKey: "title",
      meta: { title: "Title" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Title" />
      ),
      size: 150,
      minSize: 100,
      maxSize: 300,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("title")} />
      ),
    },
    {
      id: "statusCode",
      accessorKey: "statusCode",
      meta: { title: "Status" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Status" />
      ),
      size: 80,
      minSize: 60,
      maxSize: 100,
      cell: ({ row }) => {
        const statusCode = row.getValue("statusCode") as number | null
        if (!statusCode) return <span className="text-muted-foreground">-</span>
        
        let variant: "default" | "secondary" | "destructive" | "outline" = "outline"
        if (statusCode >= 200 && statusCode < 300) {
          variant = "outline"
        } else if (statusCode >= 300 && statusCode < 400) {
          variant = "secondary"
        } else if (statusCode >= 400 && statusCode < 500) {
          variant = "default"
        } else if (statusCode >= 500) {
          variant = "destructive"
        }
        
        return <Badge variant={variant} className="font-mono">{statusCode}</Badge>
      },
    },
    {
      id: "technologies",
      accessorKey: "technologies",
      meta: { title: "Tech" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Tech" />
      ),
      size: 180,
      minSize: 120,
      cell: ({ row }) => {
        const tech = row.getValue("technologies") as string[] | null
        if (!tech || tech.length === 0) return <span className="text-muted-foreground">-</span>
        return <ExpandableTagList items={tech} maxLines={2} variant="outline" />
      },
    },
    {
      id: "contentLength",
      accessorKey: "contentLength",
      meta: { title: "Length" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Length" />
      ),
      size: 100,
      minSize: 80,
      maxSize: 150,
      cell: ({ row }) => {
        const len = row.getValue("contentLength") as number | null
        if (len === null || len === undefined) return <span className="text-muted-foreground">-</span>
        return <span className="font-mono tabular-nums">{new Intl.NumberFormat().format(len)}</span>
      },
    },
    {
      id: "location",
      accessorKey: "location",
      meta: { title: "Location" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Location" />
      ),
      size: 150,
      minSize: 100,
      maxSize: 300,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("location")} />
      ),
    },
    {
      id: "webserver",
      accessorKey: "webserver",
      meta: { title: "Server" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Server" />
      ),
      size: 120,
      minSize: 80,
      maxSize: 200,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("webserver")} />
      ),
    },
    {
      id: "contentType",
      accessorKey: "contentType",
      meta: { title: "Type" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Type" />
      ),
      size: 120,
      minSize: 80,
      maxSize: 200,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("contentType")} />
      ),
    },
    {
      id: "responseBody",
      accessorKey: "responseBody",
      meta: { title: "Body" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Body" />
      ),
      size: 300,
      minSize: 200,
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("responseBody")} maxLines={3} />
      ),
    },
    {
      id: "responseHeaders",
      accessorKey: "responseHeaders",
      meta: { title: "Headers" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Headers" />
      ),
      size: 250,
      minSize: 150,
      maxSize: 400,
      cell: ({ row }) => {
        const headers = row.getValue("responseHeaders") as Record<string, string> | null
        if (!headers || Object.keys(headers).length === 0) {
          return <span className="text-muted-foreground">-</span>
        }
        const headersStr = Object.entries(headers)
          .map(([k, v]) => `${k}: ${v}`)
          .join('\n')
        return <ExpandableCell value={headersStr} maxLines={3} />
      },
    },
    {
      id: "vhost",
      accessorKey: "vhost",
      meta: { title: "VHost" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="VHost" />
      ),
      size: 80,
      minSize: 60,
      maxSize: 100,
      cell: ({ row }) => {
        const vhost = row.getValue("vhost") as boolean | null
        if (vhost === null || vhost === undefined) return <span className="text-muted-foreground">-</span>
        return <span className="font-mono text-sm">{vhost ? "true" : "false"}</span>
      },
    },
    {
      id: "createdAt",
      accessorKey: "createdAt",
      meta: { title: "Created" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Created" />
      ),
      size: 150,
      minSize: 120,
      maxSize: 200,
      cell: ({ row }) => {
        const createdAt = row.getValue("createdAt") as string | null
        if (!createdAt) return <span className="text-muted-foreground">-</span>
        return <span className="text-sm">{formatDate(createdAt)}</span>
      },
    },
  ], [formatDate])

	const columns = useMemo(() => baseColumns, [baseColumns])

  return (
    <UnifiedDataTable
      columns={columns}
      data={results}
      getRowId={(row) => String(row.id)}
      ui={{
        hideToolbar: true,
        hidePagination: true,
      }}
      behavior={{
        enableRowSelection: false,
      }}
    />
  )
}
