"use client"

import * as React from "react"
import type { ColumnDef } from "@tanstack/react-table"
import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"
import { useFingerprintTableActions } from "@/components/fingerprints/fingerprint-table-actions"
import type { FilterField } from "@/components/common/smart-filter-input"
import type { ARLFingerprint } from "@/types/fingerprint.types"
import type { PaginationInfo } from "@/types/common.types"
import { useTranslations } from "next-intl"

const ARL_FILTER_EXAMPLES = [
  'name="Apache"',
  'name=="Nginx"',
  'name="WordPress"',
]

interface ARLFingerprintDataTableProps {
  data: ARLFingerprint[]
  columns: ColumnDef<ARLFingerprint>[]
  onSelectionChange?: (selectedRows: ARLFingerprint[]) => void
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

export function ARLFingerprintDataTable({
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
}: ARLFingerprintDataTableProps) {
  const t = useTranslations("tools.fingerprints")

  const arlFilterFields: FilterField[] = React.useMemo(() => [
    { key: "name", label: "Name", description: t("filter.arl.name") },
  ], [t])

  const handleSmartSearch = (rawQuery: string) => {
    if (onFilterChange) {
      onFilterChange(rawQuery)
    }
  }

  const { handleSelectionChange, toolbarRight, dialogs } = useFingerprintTableActions<ARLFingerprint>({
    onSelectionChange,
    onAddSingle,
    onAddImport,
    onExport,
    onBulkDelete,
    onDeleteAll,
    totalCount,
    exportLabelOverride: `${t("actions.exportAll")} (YAML)`,
    importLabelOverride: `${t("actions.importFile")} (YAML)`,
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
          filterFields: arlFilterFields,
          filterExamples: ARL_FILTER_EXAMPLES,
          emptyMessage: "No results",
          toolbarRight,
        }}
      />
      {dialogs}
    </>
  )
}
