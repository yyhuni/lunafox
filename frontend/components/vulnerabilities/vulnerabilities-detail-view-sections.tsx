"use client"

import dynamic from "next/dynamic"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"
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
import { VulnerabilitiesDataTable } from "./vulnerabilities-data-table"

import type { VulnerabilitiesDetailViewState } from "./vulnerabilities-detail-view-state"

const VulnerabilityDetailDialog = dynamic(
  () => import("./vulnerability-detail-dialog").then((mod) => mod.VulnerabilityDetailDialog),
  { ssr: false }
)

export function VulnerabilitiesDetailViewLoadingState() {
  return <DataTableSkeleton toolbarButtonCount={2} rows={6} columns={6} />
}

export function VulnerabilitiesDetailViewContent({
  state,
  hideToolbar,
}: {
  state: VulnerabilitiesDetailViewState
  hideToolbar: boolean
}) {
  return (
    <>
      {state.detailDialogOpen && state.selectedVulnerability ? (
        <VulnerabilityDetailDialog
          vulnerability={state.selectedVulnerability}
          open={state.detailDialogOpen}
          onOpenChange={state.setDetailDialogOpen}
        />
      ) : null}

      <VulnerabilitiesDataTable
        data={state.vulnerabilities}
        columns={state.vulnerabilityColumns}
        filterValue={state.filterQuery}
        onFilterChange={state.handleFilterChange}
        pagination={state.pagination}
        setPagination={state.setPagination}
        paginationInfo={state.paginationInfo}
        onPaginationChange={state.handlePaginationChange}
        onSelectionChange={state.setSelectedVulnerabilities}
        hideToolbar={hideToolbar}
        reviewFilter={state.reviewFilter}
        onReviewFilterChange={state.handleReviewFilterChange}
        pendingCount={state.pendingCount}
        reviewedCount={state.reviewedCount}
        severityFilter={state.severityFilter}
        onSeverityFilterChange={state.handleSeverityFilterChange}
        selectedRows={state.selectedVulnerabilities}
        onBulkMarkAsReviewed={state.handleBulkMarkAsReviewed}
        onBulkMarkAsPending={state.handleBulkMarkAsPending}
      />
    </>
  )
}

export function VulnerabilitiesDetailViewDialogs({ state }: { state: VulnerabilitiesDetailViewState }) {
  return (
    <>
      <AlertDialog open={state.deleteDialogOpen} onOpenChange={state.setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {state.tConfirm("deleteVulnMessage", { name: state.vulnerabilityToDelete?.vulnType ?? "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{state.tCommon("actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={state.confirmDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {state.tCommon("actions.delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={state.bulkDeleteDialogOpen} onOpenChange={state.setBulkDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("bulkDeleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {state.tConfirm("bulkDeleteVulnMessage", { count: state.selectedVulnerabilities.length })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="mt-2 p-2 bg-muted rounded-md max-h-96 overflow-y-auto">
            <ul className="text-sm space-y-1">
              {state.selectedVulnerabilities.map((vulnerability) => (
                <li key={vulnerability.id} className="flex items-center">
                  <span className="font-medium">{vulnerability.vulnType}</span>
                </li>
              ))}
            </ul>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel>{state.tCommon("actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={state.confirmBulkDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {state.tConfirm("deleteVulnCount", { count: state.selectedVulnerabilities.length })}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
