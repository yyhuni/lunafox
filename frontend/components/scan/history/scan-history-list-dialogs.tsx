import React from "react"
import {
  AlertDialog,
  AlertDialogClose, AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import type { ScanRecord } from "@/types/scan.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface ScanHistoryDialogsProps {
  tConfirm: TranslationFn
  tCommon: TranslationFn
  deleteDialogOpen: boolean
  setDeleteDialogOpen: (open: boolean) => void
  scanToDelete: ScanRecord | null
  onConfirmDelete: () => void
  bulkDeleteDialogOpen: boolean
  setBulkDeleteDialogOpen: (open: boolean) => void
  selectedScans: ScanRecord[]
  onConfirmBulkDelete: () => void
  stopDialogOpen: boolean
  setStopDialogOpen: (open: boolean) => void
  scanToStop: ScanRecord | null
  onConfirmStop: () => void
  batchStopDialogOpen: boolean
  setBatchStopDialogOpen: (open: boolean) => void
  activeSelectedCount: number
  terminalSelectedCount: number
  onConfirmBatchStop: () => void
}

export function ScanHistoryDialogs({
  tConfirm,
  tCommon,
  deleteDialogOpen,
  setDeleteDialogOpen,
  scanToDelete,
  onConfirmDelete,
  bulkDeleteDialogOpen,
  setBulkDeleteDialogOpen,
  selectedScans,
  onConfirmBulkDelete,
  stopDialogOpen,
  setStopDialogOpen,
  scanToStop,
  onConfirmStop,
  batchStopDialogOpen,
  setBatchStopDialogOpen,
  activeSelectedCount,
  terminalSelectedCount,
  onConfirmBatchStop,
}: ScanHistoryDialogsProps) {
  return (
    <>
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{tConfirm("deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {tConfirm("deleteScanMessage", { name: scanToDelete?.target?.name ?? "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline">{tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={onConfirmDelete}
              className="bg-destructive hover:bg-destructive/90 text-destructive-foreground"
            >
              {tCommon("actions.delete")}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={bulkDeleteDialogOpen} onOpenChange={setBulkDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{tConfirm("bulkDeleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {tConfirm("bulkDeleteScanMessage", { count: selectedScans.length })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline">{tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={onConfirmBulkDelete}
              className="bg-destructive hover:bg-destructive/90 text-destructive-foreground"
            >
              {tConfirm("confirmDelete")}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={stopDialogOpen} onOpenChange={setStopDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{tConfirm("stopScanTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {tConfirm("stopScanMessage", { name: scanToStop?.target?.name ?? "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline">{tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={onConfirmStop}
            >
              {tConfirm("stopScanAction")}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={batchStopDialogOpen} onOpenChange={setBatchStopDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{tConfirm("batchStopScanTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {tConfirm("batchStopScanMessage", {
                activeCount: activeSelectedCount,
                skippedCount: terminalSelectedCount,
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline">{tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose onClick={onConfirmBatchStop}>
              {tConfirm("batchStopScanAction")}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
