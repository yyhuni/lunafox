"use client"

import * as React from "react"
import {
  IconChevronDown,
  IconTrash,
  IconDownload,
  IconUpload,
  IconPlus,
  IconSettings,
} from "@/components/icons"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
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
import { useTranslations } from "next-intl"

type UseFingerprintTableActionsProps<TData> = {
  onSelectionChange?: (selectedRows: TData[]) => void
  onAddSingle?: () => void
  onAddImport?: () => void
  onExport?: () => void
  onBulkDelete?: () => void
  onDeleteAll?: () => void
  totalCount?: number
  exportLabelOverride?: string
  importLabelOverride?: string
}

export function useFingerprintTableActions<TData>({
  onSelectionChange,
  onAddSingle,
  onAddImport,
  onExport,
  onBulkDelete,
  onDeleteAll,
  totalCount = 0,
  exportLabelOverride,
  importLabelOverride,
}: UseFingerprintTableActionsProps<TData>) {
  const [selectedCount, setSelectedCount] = React.useState(0)
  const [exportDialogOpen, setExportDialogOpen] = React.useState(false)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [deleteAllDialogOpen, setDeleteAllDialogOpen] = React.useState(false)
  const t = useTranslations("tools.fingerprints")
  const tCommon = useTranslations("common.actions")

  const exportLabel = exportLabelOverride ?? t("actions.exportAll")
  const importLabel = importLabelOverride ?? t("actions.importFile")

  const handleSelectionChange = React.useCallback((rows: TData[]) => {
    setSelectedCount(rows.length)
    onSelectionChange?.(rows)
  }, [onSelectionChange])

  const toolbarRight = (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm">
            <IconSettings className="h-4 w-4" />
            {t("actions.operations")}
            <IconChevronDown className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-48">
          {onExport && (
            <DropdownMenuItem onClick={() => setExportDialogOpen(true)}>
              <IconDownload className="h-4 w-4" />
              {exportLabel}
            </DropdownMenuItem>
          )}
          <DropdownMenuSeparator />
          {onBulkDelete && (
            <DropdownMenuItem
              onClick={() => setBulkDeleteDialogOpen(true)}
              disabled={selectedCount === 0}
              className="text-destructive focus:text-destructive"
            >
              <IconTrash className="h-4 w-4" />
              {t("actions.deleteSelected")} ({selectedCount})
            </DropdownMenuItem>
          )}
          {onDeleteAll && (
            <DropdownMenuItem
              onClick={() => setDeleteAllDialogOpen(true)}
              className="text-destructive focus:text-destructive"
            >
              <IconTrash className="h-4 w-4" />
              {t("actions.deleteAll")}
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>

      {(onAddSingle || onAddImport) && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button size="sm">
              <IconPlus className="h-4 w-4" />
              {t("actions.addFingerprint")}
              <IconChevronDown className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-40">
            {onAddSingle && (
              <DropdownMenuItem onClick={onAddSingle}>
                <IconPlus className="h-4 w-4" />
                {t("actions.addSingle")}
              </DropdownMenuItem>
            )}
            {onAddImport && (
              <DropdownMenuItem onClick={onAddImport}>
                <IconUpload className="h-4 w-4" />
                {importLabel}
              </DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </>
  )

  const dialogs = (
    <>
      <AlertDialog open={exportDialogOpen} onOpenChange={setExportDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("dialogs.exportTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("dialogs.exportDesc", { count: totalCount })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon("cancel")}</AlertDialogCancel>
            <AlertDialogAction onClick={() => { onExport?.(); setExportDialogOpen(false) }}>
              {t("dialogs.confirmExport")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={bulkDeleteDialogOpen} onOpenChange={setBulkDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("dialogs.deleteSelectedTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("dialogs.deleteSelectedDesc", { count: selectedCount })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon("cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => { onBulkDelete?.(); setBulkDeleteDialogOpen(false) }}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {t("dialogs.confirmDelete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={deleteAllDialogOpen} onOpenChange={setDeleteAllDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("dialogs.deleteAllTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("dialogs.deleteAllDesc", { count: totalCount })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon("cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => { onDeleteAll?.(); setDeleteAllDialogOpen(false) }}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {t("dialogs.confirmDelete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )

  return {
    handleSelectionChange,
    toolbarRight,
    dialogs,
  }
}
