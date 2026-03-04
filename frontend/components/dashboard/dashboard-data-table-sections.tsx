"use client"

import React from "react"

import { Skeleton } from "@/components/ui/skeleton"
import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"
import { ScanProgressDialog } from "@/components/scan/scan-progress-dialog"
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

import type { DashboardDataTableState } from "./dashboard-data-table-state"

export function DashboardDataDialogs({
  state,
}: {
  state: DashboardDataTableState
}) {
  return (
    <>
      {state.progressData ? (
        <ScanProgressDialog
          open={state.progressDialogOpen}
          onOpenChange={state.setProgressDialogOpen}
          data={state.progressData}
        />
      ) : null}

      <AlertDialog open={state.deleteDialogOpen} onOpenChange={state.setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.t("common.confirm.deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>{state.t("common.confirm.deleteMessage")}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{state.t("common.actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={state.confirmDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {state.t("common.actions.delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={state.stopDialogOpen} onOpenChange={state.setStopDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.t("common.confirm.title")}</AlertDialogTitle>
            <AlertDialogDescription>{state.t("common.confirm.deleteMessage")}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{state.t("common.actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={state.confirmStop}
              className="bg-primary text-primary-foreground hover:bg-primary/90"
            >
              {state.t("scan.stopScan")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

function DashboardTableLoading() {
  return (
    <div className="space-y-2">
      {[...Array(5)].map((_, i) => (
        <Skeleton key={i} className="h-12 w-full" />
      ))}
    </div>
  )
}

export function DashboardScanTable({
  state,
}: {
  state: DashboardDataTableState
}) {
  if (state.scanQuery.isLoading) {
    return <DashboardTableLoading />
  }

  return (
    <UnifiedDataTable
      data={state.scans}
      columns={state.scanColumns}
      getRowId={(row) => String(row.id)}
      state={{
        pagination: state.scanPagination,
        onPaginationChange: state.setScanPagination,
        paginationInfo: state.scanPaginationInfo,
      }}
      behavior={{
        enableRowSelection: false,
        enableAutoColumnSizing: true,
      }}
      actions={{
        showAddButton: false,
        showBulkDelete: false,
      }}
      ui={{
        hideToolbar: true,
        emptyMessage: state.t("common.status.noData"),
      }}
    />
  )
}
