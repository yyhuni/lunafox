"use client"

import React from "react"
import dynamic from "next/dynamic"
import { AlertTriangle, Loader2, Ban, Clock } from "@/components/icons"

import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { Skeleton } from "@/components/ui/skeleton"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
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
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"

import type { TargetSettingsState } from "./target-settings-state"

const ScheduledScanDataTable = dynamic(
  () =>
    import("@/components/scan/scheduled/scheduled-scan-data-table").then(
      (mod) => mod.ScheduledScanDataTable
    ),
  {
    loading: () => <DataTableSkeleton rows={3} columns={6} toolbarButtonCount={1} />,
  }
)

const CreateScheduledScanDialog = dynamic(
  () =>
    import("@/components/scan/scheduled/create-scheduled-scan-dialog").then(
      (mod) => mod.CreateScheduledScanDialog
    ),
  { ssr: false }
)

const EditScheduledScanDialog = dynamic(
  () =>
    import("@/components/scan/scheduled/edit-scheduled-scan-dialog").then(
      (mod) => mod.EditScheduledScanDialog
    ),
  { ssr: false }
)

export function TargetSettingsLoadingState() {
  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <Skeleton className="h-6 w-32" />
        <Skeleton className="h-4 w-96" />
      </div>
      <Skeleton className="h-48 w-full" />
      <Skeleton className="h-48 w-full" />
    </div>
  )
}

export function TargetSettingsErrorState({
  t,
}: {
  t: (key: string) => string
}) {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <AlertTriangle className="h-10 w-10 text-destructive mb-4" />
      <p className="text-muted-foreground">{t("loadError")}</p>
    </div>
  )
}

export function TargetSettingsContent({
  state,
}: {
  state: TargetSettingsState
}) {
  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Ban className="h-5 w-5 text-muted-foreground" />
            <CardTitle>{state.t("blacklist.title")}</CardTitle>
          </div>
          <CardDescription>{state.t("blacklist.description")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-muted-foreground">
            <span className="font-medium text-foreground">{state.t("blacklist.rulesTitle")}:</span>
            <span><code className="bg-muted px-1.5 py-0.5 rounded text-xs">*.gov</code> {state.t("blacklist.rules.domainShort")}</span>
            <span><code className="bg-muted px-1.5 py-0.5 rounded text-xs">*cdn*</code> {state.t("blacklist.rules.keywordShort")}</span>
            <span><code className="bg-muted px-1.5 py-0.5 rounded text-xs">192.168.1.1</code> {state.t("blacklist.rules.ipShort")}</span>
            <span><code className="bg-muted px-1.5 py-0.5 rounded text-xs">10.0.0.0/8</code> {state.t("blacklist.rules.cidrShort")}</span>
          </div>

          <Textarea
            name="blacklistRules"
            autoComplete="off"
            value={state.blacklistText}
            onChange={state.handleTextChange}
            placeholder={state.t("blacklist.placeholder")}
            className="min-h-[240px] font-mono text-sm"
          />

          <div className="flex justify-end">
            <Button onClick={state.handleSave} disabled={!state.hasChanges || state.updateBlacklist.isPending}>
              {state.updateBlacklist.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : null}
              {state.t("blacklist.save")}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Clock className="h-5 w-5 text-muted-foreground" />
            <CardTitle>{state.t("scheduledScans.title")}</CardTitle>
          </div>
          <CardDescription>{state.t("scheduledScans.description")}</CardDescription>
        </CardHeader>
        <CardContent>
          {state.isLoadingScans ? (
            <DataTableSkeleton rows={3} columns={6} toolbarButtonCount={1} />
          ) : (
            <ScheduledScanDataTable
              data={state.scheduledScans}
              columns={state.columns}
              onAddNew={state.handleAddNew}
              searchPlaceholder={state.tScan("scheduled.searchPlaceholder")}
              searchValue={state.searchQuery}
              onSearch={state.handleSearchChange}
              isSearching={state.isSearching}
              addButtonText={state.tScan("scheduled.createTitle")}
              page={state.page}
              pageSize={state.pageSize}
              total={state.scheduledPaginationInfo.total}
              totalPages={state.scheduledPaginationInfo.totalPages}
              onPageChange={state.handlePageChange}
              onPageSizeChange={state.handlePageSizeChange}
            />
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export function TargetSettingsDialogs({
  state,
}: {
  state: TargetSettingsState
}) {
  return (
    <>
      {state.createDialogOpen ? (
        <CreateScheduledScanDialog
          open={state.createDialogOpen}
          onOpenChange={state.setCreateDialogOpen}
          presetTargetId={state.targetId}
          presetTargetName={state.target?.name}
          onSuccess={() => state.refetch()}
        />
      ) : null}

      {state.editDialogOpen ? (
        <EditScheduledScanDialog
          open={state.editDialogOpen}
          onOpenChange={state.setEditDialogOpen}
          scheduledScan={state.editingScheduledScan}
          onSuccess={() => state.refetch()}
        />
      ) : null}

      <AlertDialog open={state.deleteDialogOpen} onOpenChange={state.setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {state.tConfirm("deleteScheduledScanMessage", { name: state.deletingScheduledScan?.name ?? "" })}
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
    </>
  )
}
