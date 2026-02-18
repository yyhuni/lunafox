"use client"

import * as React from "react"
import type { ColumnDef } from "@tanstack/react-table"
import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"
import { useFingerprintTableActions } from "@/components/fingerprints/fingerprint-table-actions"
import type { FilterField } from "@/components/common/smart-filter-input"
import type { EholeFingerprint } from "@/types/fingerprint.types"
import type { PaginationInfo } from "@/types/common.types"
import { useTranslations } from "next-intl"

const EHOLE_FILTER_EXAMPLES = [
  'cms="WordPress"',
  'type=="CMS"',
  'method="keyword" location="body"',
  'isImportant="true"',
]

interface EholeFingerprintDataTableProps {
  data: EholeFingerprint[]
  columns: ColumnDef<EholeFingerprint>[]
  onSelectionChange?: (selectedRows: EholeFingerprint[]) => void
  filterValue?: string
  onFilterChange?: (value: string) => void
  isSearching?: boolean
  onAddSingle?: () => void
  onAddImport?: () => void
  onExport?: () => void
  onBulkDelete?: () => void
  onDeleteAll?: () => void
  totalCount?: number
  pagination?: { pageIndex: number; pageSize: number }
  paginationInfo?: PaginationInfo
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
}

export function EholeFingerprintDataTable({
  data = [],
  columns,
  onSelectionChange,
  filterValue,
  onFilterChange,
  isSearching = false,
  onAddSingle,
  onAddImport,
  onExport,
  onBulkDelete,
  onDeleteAll,
  totalCount = 0,
  pagination: externalPagination,
  paginationInfo,
  onPaginationChange,
}: EholeFingerprintDataTableProps) {
  const t = useTranslations("tools.fingerprints")

  // EHole filter field configuration (using translations)
  const eholeFilterFields: FilterField[] = React.useMemo(() => [
    { key: "cms", label: "CMS", description: t("filter.ehole.cms") },
    { key: "method", label: "Method", description: t("filter.ehole.method") },
    { key: "location", label: "Location", description: t("filter.ehole.location") },
    { key: "type", label: "Type", description: t("filter.ehole.type") },
    { key: "isImportant", label: "Important", description: t("filter.ehole.isImportant") },
  ], [t])

  const handleSmartSearch = (rawQuery: string) => {
    if (onFilterChange) {
      onFilterChange(rawQuery)
    }
  }

  const { handleSelectionChange, toolbarRight, dialogs } = useFingerprintTableActions<EholeFingerprint>({
    onSelectionChange,
    onAddSingle,
    onAddImport,
    onExport,
    onBulkDelete,
    onDeleteAll,
    totalCount,
  })

  return (
    <>
      <UnifiedDataTable
        data={data}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          pagination: externalPagination,
          paginationInfo,
          onPaginationChange,
          searchValue: filterValue,
          isSearching,
          onSelectionChange: handleSelectionChange,
        }}
        behavior={{
          searchMode: "smart",
          onSearch: handleSmartSearch,
        }}
        actions={{
          showBulkDelete: false,
          showAddButton: false,
        }}
        ui={{
          filterFields: eholeFilterFields,
          filterExamples: EHOLE_FILTER_EXAMPLES,
          emptyMessage: "No results",
          toolbarRight,
        }}
      />
      {dialogs}
    </>
  )
}
