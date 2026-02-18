"use client"

import * as React from "react"
import type { ColumnDef } from "@tanstack/react-table"
import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"
import { useFingerprintTableActions } from "@/components/fingerprints/fingerprint-table-actions"
import type { FilterField } from "@/components/common/smart-filter-input"
import type { FingersFingerprint } from "@/types/fingerprint.types"
import type { PaginationInfo } from "@/types/common.types"
import { useTranslations } from "next-intl"

const FINGERS_FILTER_EXAMPLES = [
  'name="Apache"',
  'focus="true"',
  'link="http"',
  'name="nginx" focus="true"',
]

interface FingersFingerprintDataTableProps {
  data: FingersFingerprint[]
  columns: ColumnDef<FingersFingerprint>[]
  onSelectionChange?: (selectedRows: FingersFingerprint[]) => void
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

export function FingersFingerprintDataTable({
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
}: FingersFingerprintDataTableProps) {
  const t = useTranslations("tools.fingerprints")

  // Fingers filter field configuration
  const fingersFilterFields: FilterField[] = React.useMemo(() => [
    { key: "name", label: "Name", description: t("filter.fingers.name") },
    { key: "focus", label: "Focus", description: t("filter.fingers.focus") },
    { key: "link", label: "Link", description: t("filter.fingers.link") },
  ], [t])

  const handleSmartSearch = (rawQuery: string) => {
    if (onFilterChange) {
      onFilterChange(rawQuery)
    }
  }

  const { handleSelectionChange, toolbarRight, dialogs } = useFingerprintTableActions<FingersFingerprint>({
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
          filterFields: fingersFilterFields,
          filterExamples: FINGERS_FILTER_EXAMPLES,
          emptyMessage: "No results",
          toolbarRight,
        }}
      />
      {dialogs}
    </>
  )
}
