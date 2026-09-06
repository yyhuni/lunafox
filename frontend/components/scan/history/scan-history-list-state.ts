import React from "react"
import {
  useBatchStopScans,
  useBulkDeleteScans,
  useDeleteScan,
  useLoadScanDetail,
  useStopScan,
} from "@/hooks/use-scans"
import { buildScanProgressData, type ScanProgressData } from "@/components/scan/scan-progress-dialog"
import type { ScanRecord } from "@/types/scan.types"

type UseScanHistoryActionsProps = {
  tToast?: (key: string, params?: Record<string, string | number | Date>) => string
  selectedScans: ScanRecord[]
  onSelectionClear: () => void
}

export function useScanHistoryActions({
  tToast,
  selectedScans,
  onSelectionClear,
}: UseScanHistoryActionsProps) {
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [scanToDelete, setScanToDelete] = React.useState<ScanRecord | null>(null)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [stopDialogOpen, setStopDialogOpen] = React.useState(false)
  const [scanToStop, setScanToStop] = React.useState<ScanRecord | null>(null)
  const [batchStopDialogOpen, setBatchStopDialogOpen] = React.useState(false)
  const [progressDialogOpen, setProgressDialogOpen] = React.useState(false)
  const [progressData, setProgressData] = React.useState<ScanProgressData | null>(null)
  const [runtimeDetailOpen, setRuntimeDetailOpen] = React.useState(false)
  const [scanForRuntimeDetail, setScanForRuntimeDetail] = React.useState<ScanRecord | null>(null)

  const deleteMutation = useDeleteScan()
  const bulkDeleteMutation = useBulkDeleteScans()
  const stopMutation = useStopScan()
  const batchStopMutation = useBatchStopScans()
  const loadScanDetail = useLoadScanDetail()

  const activeSelectedScans = React.useMemo(
    () => selectedScans.filter((scan) => scan.status === "pending" || scan.status === "running"),
    [selectedScans]
  )
  const terminalSelectedCount = selectedScans.length - activeSelectedScans.length
  const canBatchStop = activeSelectedScans.length > 0

  const handleDeleteScan = React.useCallback((scan: ScanRecord) => {
    setScanToDelete(scan)
    setDeleteDialogOpen(true)
  }, [])

  const confirmDelete = React.useCallback(async () => {
    if (!scanToDelete) return

    setDeleteDialogOpen(false)

    try {
      await deleteMutation.mutateAsync(scanToDelete.id)
    } finally {
      setScanToDelete(null)
    }
  }, [deleteMutation, scanToDelete])

  const handleBulkDelete = React.useCallback(() => {
    if (selectedScans.length === 0) return
    setBulkDeleteDialogOpen(true)
  }, [selectedScans.length])

  const confirmBulkDelete = React.useCallback(async () => {
    if (selectedScans.length === 0) return

    const deletedIds = selectedScans.map((scan) => scan.id)
    setBulkDeleteDialogOpen(false)

    try {
      await bulkDeleteMutation.mutateAsync(deletedIds)
      onSelectionClear()
    } catch {
      void tToast
    }
  }, [bulkDeleteMutation, onSelectionClear, selectedScans, tToast])

  const handleStopScan = React.useCallback((scan: ScanRecord) => {
    setScanToStop(scan)
    setStopDialogOpen(true)
  }, [])

  const handleBatchStop = React.useCallback(() => {
    if (!canBatchStop || batchStopMutation.isPending) return
    setBatchStopDialogOpen(true)
  }, [batchStopMutation.isPending, canBatchStop])

  const confirmBatchStop = React.useCallback(async () => {
    if (!canBatchStop || batchStopMutation.isPending || selectedScans.length === 0) return

    const scanIds = selectedScans.map((scan) => scan.id)
    setBatchStopDialogOpen(false)

    try {
      // Send terminal rows too: the server revalidates the selection and returns
      // the authoritative skipped count when a status races this confirmation.
      await batchStopMutation.mutateAsync(scanIds)
      onSelectionClear()
    } catch {
      // Keep the selection so an operator can inspect or retry the failed batch.
    }
  }, [batchStopMutation, canBatchStop, onSelectionClear, selectedScans])

  const confirmStop = React.useCallback(async () => {
    if (!scanToStop) return

    setStopDialogOpen(false)

    try {
      await stopMutation.mutateAsync(scanToStop.id)
    } finally {
      setScanToStop(null)
    }
  }, [scanToStop, stopMutation])

  const handleViewProgress = React.useCallback(async (scan: ScanRecord) => {
    try {
      const freshScan = await loadScanDetail(scan.id)
      setProgressData(buildScanProgressData(freshScan))
      setProgressDialogOpen(true)
    } catch {
      setProgressData(buildScanProgressData(scan))
      setProgressDialogOpen(true)
    }
  }, [loadScanDetail])

  const handleViewRuntimeDetail = React.useCallback((scan: ScanRecord) => {
    setScanForRuntimeDetail(scan)
    setRuntimeDetailOpen(true)
  }, [])

  return {
    selectedScans,
    deleteDialogOpen,
    setDeleteDialogOpen,
    scanToDelete,
    bulkDeleteDialogOpen,
    setBulkDeleteDialogOpen,
    stopDialogOpen,
    setStopDialogOpen,
    scanToStop,
    progressDialogOpen,
    setProgressDialogOpen,
    progressData,
    runtimeDetailOpen,
    setRuntimeDetailOpen,
    scanForRuntimeDetail,
    handleDeleteScan,
    confirmDelete,
    handleBulkDelete,
    confirmBulkDelete,
    handleStopScan,
    confirmStop,
    batchStopDialogOpen,
    setBatchStopDialogOpen,
    activeSelectedScans,
    terminalSelectedCount,
    canBatchStop,
    batchStopMutationPending: batchStopMutation.isPending,
    handleBatchStop,
    confirmBatchStop,
    handleViewProgress,
    handleViewRuntimeDetail,
  }
}
