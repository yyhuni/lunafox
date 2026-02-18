"use client"

import React, { useState, useMemo } from "react"
import dynamic from "next/dynamic"
import { AlertTriangle } from "@/components/icons"
import { toast } from "sonner"
import { useTranslations, useLocale } from "next-intl"
import {
  useFingersFingerprints,
  useBulkDeleteFingersFingerprints,
  useDeleteAllFingersFingerprints,
} from "@/hooks/use-fingerprints"
import { useSearchState } from "@/hooks/_shared/use-search-state"
import { useStablePaginationInfo } from "@/hooks/_shared/use-stable-pagination-info"
import { FingerprintService } from "@/services/fingerprint.service"
import { FingersFingerprintDataTable } from "./fingers-fingerprint-data-table"
import { createFingersFingerprintColumns } from "./fingers-fingerprint-columns"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"
import { getDateLocale } from "@/lib/date-utils"
import { downloadBlob } from "@/lib/download-utils"
import { getErrorMessage } from "@/lib/error-utils"
import type { FingersFingerprint } from "@/types/fingerprint.types"

const FingersFingerprintDialog = dynamic(
  () => import("./fingers-fingerprint-dialog").then((mod) => mod.FingersFingerprintDialog),
  { ssr: false }
)

const ImportFingerprintDialog = dynamic(
  () => import("./import-fingerprint-dialog").then((mod) => mod.ImportFingerprintDialog),
  { ssr: false }
)

export function FingersFingerprintView() {
  const tFingerprints = useTranslations("tools.fingerprints")
  const tActions = useTranslations("common.actions")
  const locale = useLocale()
  
  const [selectedFingerprints, setSelectedFingerprints] = useState<FingersFingerprint[]>([])
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: 10 })
  const [filterQuery, setFilterQuery] = useState("")
  const [addDialogOpen, setAddDialogOpen] = useState(false)
  const [importDialogOpen, setImportDialogOpen] = useState(false)

  // Query data
  const { data, isLoading, isFetching, error, refetch } = useFingersFingerprints({
    page: pagination.pageIndex + 1,
    pageSize: pagination.pageSize,
    filter: filterQuery || undefined,
  })

  // Mutations
  const bulkDeleteMutation = useBulkDeleteFingersFingerprints()
  const deleteAllMutation = useDeleteAllFingersFingerprints()

  const { isSearching, handleSearchChange: handleFilterChange } = useSearchState({
    isFetching,
    setSearchValue: setFilterQuery,
    onResetPage: () => setPagination((prev) => ({ ...prev, pageIndex: 0 })),
  })

  // Format date
  const formatDate = React.useCallback((dateString: string): string => {
    return new Date(dateString).toLocaleString(getDateLocale(locale), {
      year: "numeric",
      month: "numeric",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
    })
  }, [locale])

  // Export
  const handleExport = async () => {
    try {
      const blob = await FingerprintService.exportFingersFingerprints()
      downloadBlob(blob, `fingers-fingerprints-${Date.now()}.json`)
      toast.success(tFingerprints("toast.exportSuccess"))
    } catch (error) {
      toast.error(getErrorMessage(error) || tFingerprints("toast.exportFailed"))
    }
  }

  // Bulk delete
  const handleBulkDelete = async () => {
    if (selectedFingerprints.length === 0) return

    try {
      const ids = selectedFingerprints.map((f) => f.id)
      const result = await bulkDeleteMutation.mutateAsync(ids)
      toast.success(tFingerprints("toast.deleteSuccess", { count: result.deleted }))
      setSelectedFingerprints([])
    } catch (error) {
      toast.error(getErrorMessage(error) || tFingerprints("toast.deleteFailed"))
    }
  }

  // Delete all
  const handleDeleteAll = async () => {
    try {
      const result = await deleteAllMutation.mutateAsync()
      toast.success(tFingerprints("toast.deleteSuccess", { count: result.deleted }))
    } catch (error) {
      toast.error(getErrorMessage(error) || tFingerprints("toast.deleteFailed"))
    }
  }

  // Column definitions
  const columns = useMemo(
    () =>
      createFingersFingerprintColumns({
        formatDate,
        selectLabels: {
          selectAll: tActions("selectAll"),
          selectRow: tActions("selectRow"),
        },
      }),
    [formatDate, tActions]
  )

  // Transform data
  const fingerprints: FingersFingerprint[] = useMemo(() => {
    if (!data?.results) return []
    return data.results
  }, [data])

  // Stabilize paginationInfo reference
  const paginationInfo = useStablePaginationInfo(data, { fallbackTotalPages: 1 })

  // Error state
  if (error) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <div className="rounded-full bg-destructive/10 p-3 mb-4">
          <AlertTriangle className="h-10 w-10 text-destructive" />
        </div>
        <h3 className="text-lg font-semibold mb-2">{tFingerprints("loadFailed")}</h3>
        <p className="text-muted-foreground text-center mb-4">
          {error.message || tFingerprints("loadError")}
        </p>
        <button type="button"
          onClick={() => refetch()}
          className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
        >
          {tFingerprints("reload")}
        </button>
      </div>
    )
  }

  // Loading state
  if (isLoading && !data) {
    return <DataTableSkeleton toolbarButtonCount={3} rows={6} columns={7} />
  }

  return (
    <>
      <FingersFingerprintDataTable
        data={fingerprints}
        columns={columns}
        onSelectionChange={setSelectedFingerprints}
        filterValue={filterQuery}
        onFilterChange={handleFilterChange}
        isSearching={isSearching}
        onAddSingle={() => setAddDialogOpen(true)}
        onAddImport={() => setImportDialogOpen(true)}
        onExport={handleExport}
        onBulkDelete={handleBulkDelete}
        onDeleteAll={handleDeleteAll}
        totalCount={data?.total || 0}
        pagination={pagination}
        paginationInfo={paginationInfo}
        onPaginationChange={setPagination}
      />

      {addDialogOpen ? (
        <FingersFingerprintDialog
          open={addDialogOpen}
          onOpenChange={setAddDialogOpen}
          onSuccess={() => refetch()}
        />
      ) : null}

      {importDialogOpen ? (
        <ImportFingerprintDialog
          open={importDialogOpen}
          onOpenChange={setImportDialogOpen}
          fingerprintType="fingers"
          onSuccess={() => refetch()}
        />
      ) : null}
    </>
  )
}
