"use client"

import * as React from "react"
import { IconTrash, IconPlus } from "@/components/icons"
import { Button } from "@/components/ui/button"
import type { DeleteConfirmationConfig } from "@/types/data-table.types"
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

export interface TableActionsProps {
  selectedCount: number

  // Bulk delete
  onBulkDelete?: () => void
  bulkDeleteLabel?: string
  showBulkDelete?: boolean
  deleteConfirmation?: DeleteConfirmationConfig

  // Add operation
  onAddNew?: () => void
  onAddHover?: () => void
  addButtonLabel?: string
  showAddButton?: boolean

  // Bulk add operation
  onBulkAdd?: () => void
  bulkAddLabel?: string
  showBulkAdd?: boolean
}

export function TableActions({
  selectedCount,
  onBulkDelete,
  bulkDeleteLabel = "Delete",
  showBulkDelete = true,
  deleteConfirmation,
  onAddNew,
  onAddHover,
  addButtonLabel = "Add",
  showAddButton = true,
  onBulkAdd,
  bulkAddLabel = "Bulk Add",
  showBulkAdd = true,
}: TableActionsProps) {
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)

  // Handle delete confirmation
  const handleDeleteClick = () => {
    if (deleteConfirmation) {
      setDeleteDialogOpen(true)
    } else {
      onBulkDelete?.()
    }
  }

  const handleDeleteConfirm = () => {
    setDeleteDialogOpen(false)
    onBulkDelete?.()
  }

  return (
    <>
      {/* Bulk delete button */}
      {showBulkDelete && onBulkDelete && (
        <Button
          onClick={handleDeleteClick}
          size="sm"
          variant="outline"
          disabled={selectedCount === 0}
          className={
            selectedCount === 0
              ? "text-muted-foreground"
              : "text-destructive hover:text-destructive hover:bg-destructive/10"
          }
        >
          <IconTrash className="h-4 w-4" />
          {bulkDeleteLabel}
        </Button>
      )}

      {/* Add button */}
      {showAddButton && onAddNew && (
        <Button onClick={onAddNew} onMouseEnter={onAddHover} size="sm">
          <IconPlus className="h-4 w-4" />
          {addButtonLabel}
        </Button>
      )}

      {/* Bulk add button */}
      {showBulkAdd && onBulkAdd && (
        <Button onClick={onBulkAdd} size="sm" variant="outline">
          <IconPlus className="h-4 w-4" />
          {bulkAddLabel}
        </Button>
      )}

      {/* Delete confirmation dialog */}
      {deleteConfirmation && (
        <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>
                {deleteConfirmation.title || "Confirm Delete"}
              </AlertDialogTitle>
              <AlertDialogDescription>
                {typeof deleteConfirmation.description === 'function'
                  ? deleteConfirmation.description(selectedCount)
                  : deleteConfirmation.description || `Are you sure you want to delete ${selectedCount} selected item(s)? This action cannot be undone.`}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>
                {deleteConfirmation.cancelLabel || "Cancel"}
              </AlertDialogCancel>
              <AlertDialogAction
                onClick={handleDeleteConfirm}
                className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              >
                {deleteConfirmation.confirmLabel || "Delete"}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      )}
    </>
  )
}
