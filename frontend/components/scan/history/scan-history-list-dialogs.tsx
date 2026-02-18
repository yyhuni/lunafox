import React from "react"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
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
            <AlertDialogCancel>{tCommon("actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={onConfirmDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {tCommon("actions.delete")}
            </AlertDialogAction>
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
          <div className="mt-2 p-2 bg-muted rounded-md max-h-96 overflow-y-auto">
            <ul className="text-sm space-y-1">
              {selectedScans.map((scan) => (
                <li key={scan.id} className="flex items-center justify-between">
                  <span className="font-medium">{scan.target?.name}</span>
                  <span className="text-muted-foreground text-xs">{scan.engineNames?.join(", ") || "-"}</span>
                </li>
              ))}
            </ul>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon("actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={onConfirmBulkDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {tConfirm("deleteScanCount", { count: selectedScans.length })}
            </AlertDialogAction>
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
            <AlertDialogCancel>{tCommon("actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={onConfirmStop}
              className="bg-primary text-primary-foreground hover:bg-primary/90"
            >
              {tConfirm("stopScanAction")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
