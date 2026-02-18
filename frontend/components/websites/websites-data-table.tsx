"use client"

import * as React from "react"
import type { ColumnDef } from "@tanstack/react-table"
import { useTranslations } from "next-intl"
import { SmartFilterDataTable } from "@/components/ui/data-table/smart-filter-data-table"
import { buildDownloadOptions } from "@/components/ui/data-table/data-table-helpers"
import type { FilterField } from "@/components/common/smart-filter-input"
import type { WebSite } from "@/types/website.types"
import type { PaginationInfo } from "@/types/common.types"

// Website page filter field configuration
const WEBSITE_FILTER_FIELDS: FilterField[] = [
  { key: "url", label: "URL", description: "Full URL" },
  { key: "host", label: "Host", description: "Hostname" },
  { key: "title", label: "Title", description: "Page title" },
  { key: "status", label: "Status", description: "HTTP status code" },
  { key: "tech", label: "Tech", description: "Technologies" },
  { key: "responseHeaders", label: "Headers", description: "Response headers" },
]

// Website page filter examples
const WEBSITE_FILTER_EXAMPLES = [
  'host="api.example.com" && status="200"',
  'title="Login" || title="Admin"',
  'url="/api/" && status!="404"',
  'tech="nginx" || tech="apache"',
]

interface WebSitesDataTableProps {
  data: WebSite[]
  columns: ColumnDef<WebSite>[]
  // Smart filter
  filterValue?: string
  onFilterChange?: (value: string) => void
  isSearching?: boolean
  pagination?: { pageIndex: number; pageSize: number }
  setPagination?: React.Dispatch<React.SetStateAction<{ pageIndex: number; pageSize: number }>>
  paginationInfo?: PaginationInfo
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
  onBulkDelete?: () => void
  onSelectionChange?: (selectedRows: WebSite[]) => void
  onDownloadAll?: () => void
  onDownloadSelected?: () => void
  onBulkAdd?: () => void
}

export function WebSitesDataTable({
  data = [],
  columns,
  filterValue,
  onFilterChange,
  isSearching = false,
  pagination,
  setPagination,
  paginationInfo,
  onPaginationChange,
  onBulkDelete,
  onSelectionChange,
  onDownloadAll,
  onDownloadSelected,
  onBulkAdd,
}: WebSitesDataTableProps) {
  const t = useTranslations("common.status")
  const tActions = useTranslations("common.actions")
  const tDownload = useTranslations("common.download")
  
  const downloadOptions = buildDownloadOptions(tDownload, {
    onDownloadAll,
    onDownloadSelected,
  })

  return (
    <SmartFilterDataTable
      data={data}
      columns={columns}
      getRowId={(row) => String(row.id)}
      filterFields={WEBSITE_FILTER_FIELDS}
      filterExamples={WEBSITE_FILTER_EXAMPLES}
      filterValue={filterValue}
      onFilterChange={onFilterChange}
      isSearching={isSearching}
      pagination={pagination}
      setPagination={setPagination}
      paginationInfo={paginationInfo}
      onPaginationChange={onPaginationChange}
      onSelectionChange={onSelectionChange}
      onBulkDelete={onBulkDelete}
      bulkDeleteLabel={tActions("delete")}
      onBulkAdd={onBulkAdd}
      bulkAddLabel={tActions("add")}
      showAddButton={false}
      downloadOptions={downloadOptions}
      emptyMessage={t("noData")}
    />
  )
}
