"use client"

import React from "react"

import { Skeleton } from "@/components/ui/skeleton"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { UnifiedDataTable } from "@/components/shared/data-table/unified-data-table"
import { ScanProgressDialog } from "@/components/scan/scan-progress-dialog"
import {
  AlertDialog,
  AlertDialogClose, AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"

import type { OverviewDataTableState } from "./overview-data-table-state"
import { normalizeError } from "@/lib/errors/normalize-error"

export function OverviewDataDialogs({
  state,
}: {
  state: OverviewDataTableState
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
            <AlertDialogClose variant="outline">{state.t("common.actions.cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={state.confirmDelete}
              className="bg-destructive hover:bg-destructive/90 text-destructive-foreground"
            >
              {state.t("common.actions.delete")}
            </AlertDialogClose>
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
            <AlertDialogClose variant="outline">{state.t("common.actions.cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={state.confirmStop}
            >
              {state.t("scan.stopScan")}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

function OverviewTableLoading() {
  return (
    <div className="space-y-2">
      {[...Array(5)].map((_, i) => (
        <Skeleton key={i} className="h-12 w-full" />
      ))}
    </div>
  )
}

export function OverviewScanTable({
  state,
}: {
  state: OverviewDataTableState
}) {
  if (state.isLoading) {
    return <OverviewTableLoading />
  }

  if (state.error) {
    return (
      <AppErrorState
        error={normalizeError(state.error, { notFoundKind: "unexpected-error" })}
        title={state.t("scan.progress.engineCatalogUnavailable")}
        onRetry={state.refetch}
        variant="section"
      />
    )
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
