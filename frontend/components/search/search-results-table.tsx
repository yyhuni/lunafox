"use client"

import { useMemo, useCallback, useState } from "react"
import { useFormatter, useTranslations } from "next-intl"
import type { ColumnDef, VisibilityState } from "@tanstack/react-table"
import { IconChevronDown, IconLayoutColumns } from "@/components/icons"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { DataTableColumnHeader } from "@/components/shared/data-table/column-header"
import { HttpStatusBadge } from "@/components/shared/status/http-status-badge"
import { UnifiedDataTable } from "@/components/shared/data-table/unified-data-table"
import { ExpandableCell, ExpandableTagList } from "@/components/shared/data-table/expandable-cell"
import { EndpointDetailDrawer } from "@/components/endpoints/endpoint-detail-drawer"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { EndpointSearchResult } from "@/types/search.types"

export const DEFAULT_SEARCH_COLUMN_VISIBILITY: VisibilityState = {
  host: false,
  location: false,
  responseBody: false,
  responseHeaders: false,
}

export interface SearchEndpointsResultModel {
  columnVisibility: VisibilityState
  setColumnVisibility: React.Dispatch<React.SetStateAction<VisibilityState>>
}

export function useSearchEndpointsResultModel(): SearchEndpointsResultModel {
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>(
    DEFAULT_SEARCH_COLUMN_VISIBILITY
  )
  return { columnVisibility, setColumnVisibility }
}

export function SearchEndpointColumnVisibilityMenu({
  model,
}: {
  model: SearchEndpointsResultModel
}) {
  const tDataTable = useTranslations("dataTable")
  const t = useTranslations("search.table")
  const columns = useMemo(() => [
    { id: "url", label: t("url") },
    { id: "host", label: t("host") },
    { id: "title", label: t("title") },
    { id: "statusCode", label: t("status") },
    { id: "tech", label: t("technologies") },
    { id: "contentLength", label: t("contentLength") },
    { id: "location", label: t("location") },
    { id: "webserver", label: t("webserver") },
    { id: "contentType", label: t("contentType") },
    { id: "responseBody", label: t("responseBody") },
    { id: "responseHeaders", label: t("responseHeaders") },
    { id: "vhost", label: t("vhost") },
    { id: "createdAt", label: t("createdAt") },
  ], [t])

  const toggleColumn = useCallback((columnId: string, value: boolean) => {
    model.setColumnVisibility((prev) => ({
      ...prev,
      [columnId]: value,
    }))
  }, [model])

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button variant="outline" size="default" className="shrink-0 gap-1.5">
            <IconLayoutColumns className="h-4 w-4" />
            {tDataTable("showColumns")}
            <IconChevronDown className="h-4 w-4" />
          </Button>
        }
      />
      <DropdownMenuContent align="end" width="content-fit">
        {columns.map((column) => {
          const isVisible = model.columnVisibility[column.id] ?? true
          return (
            <DropdownMenuCheckboxItem
              key={column.id}
              className="capitalize"
              checked={isVisible}
              onCheckedChange={(checked) => toggleColumn(column.id, Boolean(checked))}
            >
              {column.label}
            </DropdownMenuCheckboxItem>
          )
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

const EMPTY_RESULTS: EndpointSearchResult[] = []

interface SearchResultsTableProps {
  results: EndpointSearchResult[]
  model?: SearchEndpointsResultModel
  loading?: boolean
  loadingRowCount?: number
}

export function SearchResultsTable({
  results,
  model,
  loading = false,
  loadingRowCount = 10,
}: SearchResultsTableProps) {
  const [activeEndpoint, setActiveEndpoint] = useState<EndpointSearchResult | null>(null)
  const [internalColumnVisibility, setInternalColumnVisibility] = useState<VisibilityState>(
    DEFAULT_SEARCH_COLUMN_VISIBILITY
  )
  const columnVisibility = model ? model.columnVisibility : internalColumnVisibility
  const setColumnVisibility = model ? model.setColumnVisibility : setInternalColumnVisibility
  const format = useFormatter()
  const t = useTranslations("search.table")
  const tActions = useTranslations("common.actions")

  const handleSelectEndpoint = useCallback((row: unknown) => {
    setActiveEndpoint(row as EndpointSearchResult)
  }, [])

  const handleEndpointDetailOpenChange = useCallback((open: boolean) => {
    if (!open) {
      setActiveEndpoint(null)
    }
  }, [])

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
    <>
      <UnifiedDataTable
        columns={columns}
        data={results}
        getRowId={(row) => String(row.id)}
        state={{ columnVisibility, onColumnVisibilityChange: setColumnVisibility }}
        ui={{
          hideToolbar: true,
          showColumnVisibility: false,
          hidePagination: true,
          loading,
          loadingPresentation: "initial",
          loadingRowCount,
        }}
        behavior={{
          enableRowSelection: false,
          columnLayout: "fixed",
          expandColumnIds: ["url", "title"],
          onRowClick: handleSelectEndpoint,
          getRowActionLabel: (row) => `${tActions("details")}: ${(row as EndpointSearchResult).url}`,
        }}
      />
      <EndpointDetailDrawer
        endpoint={activeEndpoint}
        open={Boolean(activeEndpoint)}
        onOpenChange={handleEndpointDetailOpenChange}
        formatDate={formatDate}
      />
    </>
  )
}

export function SearchResultsTableLoadingState({
  rowCount = 10,
}: {
  rowCount?: number
}) {
  return (
    <SearchResultsTable
      results={EMPTY_RESULTS}
      loading
      loadingRowCount={rowCount}
    />
  )
}
