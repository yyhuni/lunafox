"use client"

import { IconPlus, IconRefresh } from "@/components/icons"
import { RefreshSpinner } from "@/components/shared/loading/spinner"
import { useRefreshFeedback } from "@/components/shared/loading/use-refresh-feedback"
import { Button } from "@/components/ui/button"
import type { DataTableToolbarDensity } from "@/types/data-table.types"

export interface TableActionsProps {
  selectedCount: number

  // Add operation
  onAddNew?: () => void
  onAddHover?: () => void
  addButtonLabel?: string
  showAddButton?: boolean

  // Bulk add operation
  onBulkAdd?: () => void
  bulkAddLabel?: string
  showBulkAdd?: boolean

  // Refresh operation
  onRefresh?: () => void
  refreshLabel?: string
  showRefresh?: boolean
  isRefreshing?: boolean
  toolbarDensity?: DataTableToolbarDensity
}

export function TableActions({
  onAddNew,
  onAddHover,
  addButtonLabel = "Add",
  showAddButton = true,
  onBulkAdd,
  bulkAddLabel = "Bulk Add",
  showBulkAdd = true,
  onRefresh,
  refreshLabel = "Refresh",
  showRefresh = true,
  isRefreshing = false,
  toolbarDensity = "compact",
}: TableActionsProps) {
  const actionButtonSize = toolbarDensity === "standard" ? "default" : "sm"
  const refreshButtonSize = toolbarDensity === "standard" ? "icon" : "icon-sm"
  // Keep a fast refetch visible long enough for the browser to paint feedback.
  const { isVisible: isRefreshFeedbackVisible, beginManualRefresh } = useRefreshFeedback(isRefreshing)

  return (
    <>
      {/* Refresh button */}
      {showRefresh && onRefresh && (
        <Button
          onClick={() => {
            beginManualRefresh()
            onRefresh()
          }}
          size={refreshButtonSize}
          variant="outline"
          disabled={isRefreshFeedbackVisible}
          aria-label={refreshLabel}
          aria-busy={isRefreshFeedbackVisible}
        >
          {isRefreshFeedbackVisible ? (
            <RefreshSpinner className="h-4 w-4" />
          ) : (
            <IconRefresh className="h-4 w-4" />
          )}
          <span className="sr-only">{refreshLabel}</span>
        </Button>
      )}

      {/* Add button */}
      {showAddButton && onAddNew && (
        <Button onClick={onAddNew} onMouseEnter={onAddHover} onFocus={onAddHover} size={actionButtonSize}>
          <IconPlus className="h-4 w-4" />
          {addButtonLabel}
        </Button>
      )}

      {/* Bulk add button */}
      {showBulkAdd && onBulkAdd && (
        <Button onClick={onBulkAdd} size={actionButtonSize} variant="outline">
          <IconPlus className="h-4 w-4" />
          {bulkAddLabel}
        </Button>
      )}
    </>
  )
}
