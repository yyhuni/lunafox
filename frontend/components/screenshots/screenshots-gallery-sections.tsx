"use client"

import React from "react"
import {
  Image as ImageIcon,
  ExternalLink,
  Trash2,
  X,
  ChevronLeft,
  ChevronRight,
} from "@/components/icons"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { ConfirmDialog } from "@/components/shared/feedback/confirm-dialog"
import { HttpStatusBadge } from "@/components/shared/status/http-status-badge"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { SearchToolbarSkeleton } from "@/components/shared/loading/search-toolbar-skeleton"
import { Skeleton } from "@/components/ui/skeleton"
import {
  DataTableFacetedFilter,
  DataTableFacetedFilterGroup,
  type DataTableFacetedFilterOption,
} from "@/components/shared/data-table/faceted-filter"
import {
  SelectedRowActionBar,
  type SelectedRowActionBarAction,
} from "@/components/shared/data-table"
import { SharedCompactPagination } from "@/components/shared/data-table/pagination"
import { SimpleSearchToolbar } from "@/components/shared/data-table/simple-search-toolbar"
import {
  SCREENSHOTS_GALLERY_GRID_CLASS,
  SCREENSHOTS_GALLERY_ROOT_CLASS,
  SCREENSHOTS_GALLERY_TOOLBAR_ACTIONS_CLASS,
  SCREENSHOTS_GALLERY_TOOLBAR_CLASS,
  SCREENSHOTS_GALLERY_TOOLBAR_CONTROLS_CLASS,
} from "./screenshots-gallery-layout"
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import { ScreenshotSortControl } from "./screenshot-sort-control"

import type { ScreenshotsGalleryState } from "./screenshots-gallery-state"

const SCREENSHOT_GALLERY_LOADING_ITEM_COUNT = 8

function mergeSelectedFilterOptions(
  options: Array<DataTableFacetedFilterOption<string>>,
  selected: string[]
) {
  const optionByValue = new Map(options.map((option) => [option.value, option]))
  for (const value of selected) {
    const trimmed = value.trim()
    if (trimmed && !optionByValue.has(trimmed)) {
      optionByValue.set(trimmed, { value: trimmed, label: trimmed })
    }
  }
  return Array.from(optionByValue.values()).sort((left, right) =>
    left.label.localeCompare(right.label, undefined, { numeric: true })
  )
}

function ScreenshotsGalleryToolbarLoadingState({
  showSelectionAction = false,
}: {
  showSelectionAction?: boolean
} = {}) {
  return (
    <div className={SCREENSHOTS_GALLERY_TOOLBAR_CLASS}>
      <div className={SCREENSHOTS_GALLERY_TOOLBAR_CONTROLS_CLASS}>
        <SearchToolbarSkeleton
          toolbarDensity="compact"
          placeholderWidthClassName="w-28"
        />
        <ActionSkeleton size="sm" widthClassName="w-24" />
        <div className="flex items-center gap-1">
          <ActionSkeleton size="sm" widthClassName="w-24" />
          <ActionSkeleton size="sm" widthClassName="w-28" />
        </div>
      </div>
      {showSelectionAction ? (
        <div className={SCREENSHOTS_GALLERY_TOOLBAR_ACTIONS_CLASS}>
          <ActionSkeleton size="sm" widthClassName="w-16" />
        </div>
      ) : null}
    </div>
  )
}

function ScreenshotsGalleryItemLoadingState() {
  return <Skeleton className="aspect-video rounded-lg" />
}

export function ScreenshotsGalleryLoadingState({
  state,
}: {
  state: ScreenshotsGalleryState
}) {
  return (
    <div className={SCREENSHOTS_GALLERY_ROOT_CLASS}>
      <ScreenshotsGalleryToolbarLoadingState />
      <div className={SCREENSHOTS_GALLERY_GRID_CLASS}>
        {Array.from({ length: Math.min(SCREENSHOT_GALLERY_LOADING_ITEM_COUNT, state.pagination.pageSize) }).map((_, index) => (
          <ScreenshotsGalleryItemLoadingState key={index} />
        ))}
      </div>
    </div>
  )
}

export function ScreenshotsGalleryRouteFallback({
  itemCount = SCREENSHOT_GALLERY_LOADING_ITEM_COUNT,
  showSelectionAction = false,
}: { itemCount?: number; showSelectionAction?: boolean } = {}) {
  if (!Number.isInteger(itemCount) || itemCount <= 0) {
    throw new Error("ScreenshotsGalleryRouteFallback itemCount must be a positive integer.")
  }

  return (
    <div className={SCREENSHOTS_GALLERY_ROOT_CLASS}>
      <ScreenshotsGalleryToolbarLoadingState showSelectionAction={showSelectionAction} />
      <div className={SCREENSHOTS_GALLERY_GRID_CLASS}>
        {Array.from({ length: itemCount }).map((_, index) => (
          <ScreenshotsGalleryItemLoadingState key={index} />
        ))}
      </div>
    </div>
  )
}

export function ScreenshotsGalleryEmptyState({
  state,
}: {
  state: ScreenshotsGalleryState
}) {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <div className="bg-muted mb-4 p-3 rounded-full">
        <ImageIcon className="h-10 text-muted-foreground w-10" />
      </div>
      <h3 className={cn("mb-2", textRole.panelTitle)}>{state.t("empty.title")}</h3>
      <p className={cn("text-center", textRole.bodySubtle)}>{state.t("empty.description")}</p>
    </div>
  )
}

export function ScreenshotsGalleryContent({
  state,
}: {
  state: ScreenshotsGalleryState
}) {
  const lightboxScreenshot = state.screenshots[state.lightboxIndex]
  const tActions = useTranslations("common.actions")
  const tColumns = useTranslations("columns")
  const tStatus = useTranslations("common.status")
  const tDataTable = useTranslations("dataTable")
  const tPagination = useTranslations("common.pagination")
  const mergedStatusCodeOptions = React.useMemo(
    () => mergeSelectedFilterOptions(state.statusCodeOptions, state.statusCodeFilter),
    [state.statusCodeFilter, state.statusCodeOptions]
  )
  const hasSelectedFilters = state.statusCodeFilter.length > 0
  const activeSort = state.sorting[0]
  const selectedRowActions: SelectedRowActionBarAction[] = state.targetId
    ? [{
        key: "delete",
        label: tActions("delete"),
        icon: Trash2,
        tone: "destructive",
        group: "danger",
        onClick: () => state.setDeleteDialogOpen(true),
      }]
    : []
  const handleSort = (field: "statusCode" | "createdAt") => {
    const isActive = activeSort?.id === field
    const desc = isActive ? !activeSort.desc : field === "createdAt"
    state.handleSortingChange([{ id: field, desc }])
  }

  return (
    <div className={SCREENSHOTS_GALLERY_ROOT_CLASS}>
      <div className={SCREENSHOTS_GALLERY_TOOLBAR_CLASS}>
        <div className={SCREENSHOTS_GALLERY_TOOLBAR_CONTROLS_CLASS}>
          <SimpleSearchToolbar
            value={state.filterQuery}
            onChange={state.commitFilterSearch}
            onSubmit={() => state.commitFilterSearch(state.filterQuery)}
            loading={state.isSearching}
            placeholder={tActions("searchURL")}
            toolbarDensity="compact"
          />
          <DataTableFacetedFilterGroup
            hasSelectedValues={hasSelectedFilters}
            onReset={() => state.handleStatusCodeFilterChange([])}
          >
            <DataTableFacetedFilter
              title={tColumns("website.statusCode")}
              values={state.statusCodeFilter}
              onValuesChange={state.handleStatusCodeFilterChange}
              options={mergedStatusCodeOptions}
              emptyLabel={tStatus("noData")}
              clearLabel={tDataTable("clearFilter")}
            />
          </DataTableFacetedFilterGroup>
          <ScreenshotSortControl
            activeSort={activeSort}
            onSort={handleSort}
            statusCodeLabel={tColumns("website.statusCode")}
            createdAtLabel={tColumns("common.createdAt")}
          />
        </div>
        <div className={SCREENSHOTS_GALLERY_TOOLBAR_ACTIONS_CLASS}>
          {state.screenshots.length > 0 && state.targetId && state.selectedIds.size === 0 ? (
            <Button variant="outline" size="sm" onClick={state.selectAll}>
              {state.tCommon("actions.selectAll")}
            </Button>
          ) : null}
        </div>
      </div>

      <div className={SCREENSHOTS_GALLERY_GRID_CLASS}>
        {state.screenshots.map((screenshot, index) => (
          <div
            key={screenshot.id}
            className={cn(
              "group relative aspect-video rounded-lg overflow-hidden border bg-muted transition-[background-color,border-color,box-shadow]",
              state.selectedIds.has(screenshot.id) ? "ring-2 ring-primary" : ""
            )}
          >
            {state.targetId ? (
              <Checkbox
                checked={state.selectedIds.has(screenshot.id)}
                onCheckedChange={() => state.toggleSelect(screenshot.id)}
                onClick={(event) => event.stopPropagation()}
                aria-label={state.t("selectScreenshot", { index: index + 1 })}
                className={cn("absolute left-2 top-2 z-10 backdrop-blur-sm bg-background/80")}
              />
            ) : null}

            <button
              type="button"
              onClick={() => state.openLightbox(index)}
              className="h-full w-full"
              aria-label={state.t("openScreenshot", { index: index + 1 })}
            >
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={state.getImageUrl(screenshot)}
                alt={screenshot.url}
                className="group-hover:scale-105 h-full object-cover transition-transform w-full"
                loading="lazy"
                width={1280}
                height={720}
              />
            </button>

            <div className="absolute bg-gradient-to-t bottom-0 from-[var(--media-overlay-gradient)] inset-x-0 p-2 to-transparent">
              <div className="flex gap-2 items-center">
                {screenshot.statusCode ? (
                  <HttpStatusBadge
                    statusCode={screenshot.statusCode}
                    size="overlay"
                    className="shrink-0"
                  />
                ) : null}
                <p className="text-[var(--media-overlay-foreground)] text-xs truncate" title={screenshot.url}>
                  {screenshot.url}
                </p>
              </div>
            </div>

            <div className="absolute group-hover:opacity-100 opacity-0 right-2 top-2 transition-opacity">
              <a
                href={screenshot.url}
                target="_blank"
                rel="noopener noreferrer"
                onClick={(e) => e.stopPropagation()}
                className="backdrop-blur-sm bg-background/80 h-8 hover:bg-background inline-flex items-center justify-center rounded-md w-8"
                aria-label={state.t("openInNewTab")}
              >
                <ExternalLink className="h-4 w-4" />
              </a>
            </div>
          </div>
        ))}
      </div>

      {state.screenshots.length === 0 && (state.filterQuery || state.statusCodeFilter.length > 0) ? (
        <div className="flex flex-col items-center justify-center py-12">
          <p className="text-muted-foreground">{state.t("noResults")}</p>
        </div>
      ) : null}

      <SharedCompactPagination
        mode="cursor"
        pageSize={state.pagination.pageSize}
        canFirstPage={state.paginationNavigation.canFirstPage}
        canPreviousPage={state.paginationNavigation.canPreviousPage}
        canNextPage={state.paginationNavigation.canNextPage}
        onFirstPage={() => state.handlePaginationChange({ pageIndex: 0, pageSize: state.pagination.pageSize })}
        onPreviousPage={() => state.handlePaginationChange({
          pageIndex: state.pagination.pageIndex - 1,
          pageSize: state.pagination.pageSize,
        })}
        onNextPage={() => state.handlePaginationChange({
          pageIndex: state.pagination.pageIndex + 1,
          pageSize: state.pagination.pageSize,
        })}
        onPageSizeChange={(pageSize) => state.handlePaginationChange({ pageIndex: 0, pageSize })}
        pageSizeOptions={state.pageSizeOptions}
        summary={tPagination("total", { count: state.cursorPaginationSummary.total })}
      />

      {state.targetId ? (
        <SelectedRowActionBar
          selectedCount={state.selectedIds.size}
          ariaLabel={tDataTable("selected", { count: state.selectedIds.size })}
          countLabel={tDataTable("selected", { count: state.selectedIds.size })}
          actions={selectedRowActions}
          onClearSelection={state.clearSelection}
          clearSelectionLabel={tActions("deselectAll")}
        />
      ) : null}

      <Dialog open={state.lightboxOpen} onOpenChange={state.setLightboxOpen}>
        <DialogContent className="bg-[var(--media-overlay-background)] border-none max-h-[90vh] max-w-[90vw] p-0">
          <DialogTitle className="sr-only">{state.t("lightboxTitle")}</DialogTitle>
          <div className="flex h-full items-center justify-center relative w-full">
            <Button
              type="button"
              variant="ghost"
              size="icon-lg"
              onClick={() => state.setLightboxOpen(false)}
              className="absolute bg-background/10 hover:bg-background/20 right-4 rounded-full top-4 z-50"
              aria-label={state.t("closeLightbox")}
            >
              <X className="h-6 text-background w-6" />
            </Button>

            {state.screenshots.length > 1 ? (
              <>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-lg"
                  onClick={state.prevImage}
                  className="absolute bg-background/10 hover:bg-background/20 left-4 rounded-full z-50"
                  aria-label={state.t("previousScreenshot")}
                >
                  <ChevronLeft className="h-8 text-background w-8" />
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-lg"
                  onClick={state.nextImage}
                  className="absolute bg-background/10 hover:bg-background/20 right-4 rounded-full z-50"
                  aria-label={state.t("nextScreenshot")}
                >
                  <ChevronRight className="h-8 text-background w-8" />
                </Button>
              </>
            ) : null}

            {lightboxScreenshot ? (
              <div className="flex flex-col gap-4 items-center p-8">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src={state.getImageUrl(lightboxScreenshot)}
                  alt={lightboxScreenshot.url}
                  className="max-h-[70vh] max-w-full object-contain"
                  width={1920}
                  height={1080}
                />
                <div className="text-center text-[var(--media-overlay-foreground)]">
                  <p className="opacity-80 text-sm">
                    {state.lightboxIndex + 1} / {state.screenshots.length}
                  </p>
                  <div className="flex gap-2 items-center justify-center mt-1">
                    {lightboxScreenshot.statusCode ? (
                      <HttpStatusBadge
                        statusCode={lightboxScreenshot.statusCode}
                        size="overlay"
                      />
                    ) : null}
                    <a
                      href={lightboxScreenshot.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex gap-1 hover:underline items-center text-info"
                    >
                      {lightboxScreenshot.url}
                      <ExternalLink className="h-3 w-3" />
                    </a>
                  </div>
                </div>
              </div>
            ) : null}
          </div>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={state.deleteDialogOpen}
        onOpenChange={state.setDeleteDialogOpen}
        title={state.tCommon("actions.confirmDelete")}
        description={state.tCommon("actions.deleteConfirmMessage", { count: state.selectedIds.size })}
        onConfirm={state.handleBulkDelete}
        loading={state.isDeleting}
        variant="destructive"
      />
    </div>
  )
}
