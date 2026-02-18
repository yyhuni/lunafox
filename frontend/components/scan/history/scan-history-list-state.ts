import React from "react"
import { toast } from "sonner"
import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import { deleteScan, bulkDeleteScans, stopScan, getScan } from "@/services/scan.service"
import { buildScanProgressData, type ScanProgressData } from "@/components/scan/scan-progress-dialog"
import type { ScanRecord } from "@/types/scan.types"

type UseScanHistoryActionsProps = {
  tToast: (key: string, params?: Record<string, string | number | Date>) => string
}

export function useScanHistoryActions({ tToast }: UseScanHistoryActionsProps) {
  const [selectedScans, setSelectedScans] = React.useState<ScanRecord[]>([])
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [scanToDelete, setScanToDelete] = React.useState<ScanRecord | null>(null)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [stopDialogOpen, setStopDialogOpen] = React.useState(false)
  const [scanToStop, setScanToStop] = React.useState<ScanRecord | null>(null)
  const [progressDialogOpen, setProgressDialogOpen] = React.useState(false)
  const [progressData, setProgressData] = React.useState<ScanProgressData | null>(null)

  const deleteMutation = useResourceMutation({
    mutationFn: deleteScan,
    invalidate: [{ queryKey: ["scans"] }],
    skipDefaultErrorHandler: true,
  })

  const bulkDeleteMutation = useResourceMutation({
    mutationFn: bulkDeleteScans,
    invalidate: [{ queryKey: ["scans"] }],
    onSuccess: () => {
      setSelectedScans([])
    },
    skipDefaultErrorHandler: true,
  })

  const stopMutation = useResourceMutation({
    mutationFn: stopScan,
    invalidate: [{ queryKey: ["scans"] }],
    skipDefaultErrorHandler: true,
  })

  const handleDeleteScan = React.useCallback((scan: ScanRecord) => {
    setScanToDelete(scan)
    setDeleteDialogOpen(true)
  }, [])

  const confirmDelete = React.useCallback(async () => {
    if (!scanToDelete) return

    setDeleteDialogOpen(false)

    try {
      await deleteMutation.mutateAsync(scanToDelete.id)
      toast.success(tToast("deletedScanRecord", { name: scanToDelete.target?.name ?? "" }))
    } catch {
      toast.error(tToast("deleteFailed"))
    } finally {
      setScanToDelete(null)
    }
  }, [deleteMutation, scanToDelete, tToast])

  const handleBulkDelete = React.useCallback(() => {
    if (selectedScans.length === 0) return
    setBulkDeleteDialogOpen(true)
  }, [selectedScans.length])

  const confirmBulkDelete = React.useCallback(async () => {
    if (selectedScans.length === 0) return

    const deletedIds = selectedScans.map((scan) => scan.id)
    setBulkDeleteDialogOpen(false)

    try {
      const result = await bulkDeleteMutation.mutateAsync(deletedIds)
      toast.success(result.message || tToast("bulkDeleteSuccess", { count: result.deletedCount }))
    } catch {
      toast.error(tToast("bulkDeleteFailed"))
    }
  }, [bulkDeleteMutation, selectedScans, tToast])

  const handleStopScan = React.useCallback((scan: ScanRecord) => {
    setScanToStop(scan)
    setStopDialogOpen(true)
  }, [])

  const confirmStop = React.useCallback(async () => {
    if (!scanToStop) return

    setStopDialogOpen(false)

    try {
      await stopMutation.mutateAsync(scanToStop.id)
      toast.success(tToast("stoppedScan", { name: scanToStop.target?.name ?? "" }))
    } catch {
      toast.error(tToast("stopFailed"))
    } finally {
      setScanToStop(null)
    }
  }, [scanToStop, stopMutation, tToast])

  const handleViewProgress = React.useCallback(async (scan: ScanRecord) => {
    try {
      const freshScan = await getScan(scan.id)
      setProgressData(buildScanProgressData(freshScan))
      setProgressDialogOpen(true)
    } catch {
      setProgressData(buildScanProgressData(scan))
      setProgressDialogOpen(true)
    }
  }, [])

  return {
    selectedScans,
    setSelectedScans,
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
    handleDeleteScan,
    confirmDelete,
    handleBulkDelete,
    confirmBulkDelete,
    handleStopScan,
    confirmStop,
    handleViewProgress,
  }
}
