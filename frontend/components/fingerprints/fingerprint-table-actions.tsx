"use client"

import * as React from "react"
import {
  IconTrash,
  IconFileExport,
  IconUpload,
  IconSettings,
} from "@/components/icons"
import {
  DropdownMenuItem,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu"
import {
  AlertDialog,
  AlertDialogClose, AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { useTranslations } from "next-intl"
import {
  ToolbarActionMenu,
  type SelectedRowActionBarAction,
} from "@/components/shared/data-table"
import { Button } from "@/components/ui/button"

type UseFingerprintTableActionsProps<TData> = {
  onSelectionChange?: (selectedRows: TData[]) => void
  onImport?: () => void
  onExport?: () => void
  onBulkDelete?: () => void
  onDeleteAll?: () => void
  totalCount?: number
  exportLabelOverride?: string
  importLabelOverride?: string
}

export function useFingerprintTableActions<TData>({
  onSelectionChange,
  onImport,
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
      <ToolbarActionMenu
        label={t("actions.operations")}
        icon={<IconSettings className="h-4 w-4" />}
        align="start"
      >
        {onExport && (
          <DropdownMenuItem onClick={() => setExportDialogOpen(true)}>
            <IconFileExport className="h-4 w-4" />
            {exportLabel}
          </DropdownMenuItem>
        )}
        {onDeleteAll && <DropdownMenuSeparator />}
        {onDeleteAll && (
          <DropdownMenuItem
            onClick={() => setDeleteAllDialogOpen(true)}
            className="focus:text-destructive text-destructive"
          >
            <IconTrash className="h-4 w-4" />
            {t("actions.deleteAll")}
          </DropdownMenuItem>
        )}
      </ToolbarActionMenu>

      {onImport ? (
        <Button type="button" size="sm" onClick={onImport}>
          <IconUpload className="h-4 w-4" />
          {importLabel}
        </Button>
      ) : null}
    </>
  )

  const selectedRowActions: SelectedRowActionBarAction[] = onBulkDelete
    ? [{
        key: "delete",
        label: tCommon("delete"),
        icon: IconTrash,
        tone: "destructive",
        group: "danger",
        onClick: () => setBulkDeleteDialogOpen(true),
      }]
    : []

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
            <AlertDialogClose variant="outline">{tCommon("cancel")}</AlertDialogClose>
            <AlertDialogClose onClick={() => { onExport?.(); setExportDialogOpen(false) }}>
              {t("dialogs.confirmExport")}
            </AlertDialogClose>
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
            <AlertDialogClose variant="outline">{tCommon("cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={() => { onBulkDelete?.(); setBulkDeleteDialogOpen(false) }}
              className="bg-destructive hover:bg-destructive/90 text-destructive-foreground"
            >
              {t("dialogs.confirmDelete")}
            </AlertDialogClose>
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
            <AlertDialogClose variant="outline">{tCommon("cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={() => { onDeleteAll?.(); setDeleteAllDialogOpen(false) }}
              className="bg-destructive hover:bg-destructive/90 text-destructive-foreground"
            >
              {t("dialogs.confirmDelete")}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )

  return {
    handleSelectionChange,
    selectedRowActions,
    toolbarRight,
    dialogs,
  }
}
