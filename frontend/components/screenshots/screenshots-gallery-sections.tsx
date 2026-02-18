"use client"

import React from "react"
import {
  AlertTriangle,
  Image as ImageIcon,
  ExternalLink,
  Trash2,
  X,
  ChevronLeft,
  ChevronRight,
  Search,
} from "@/components/icons"
import { VisuallyHidden } from "@radix-ui/react-visually-hidden"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Skeleton } from "@/components/ui/skeleton"
import { ConfirmDialog } from "@/components/ui/confirm-dialog"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog"
import { cn } from "@/lib/utils"

import type { ScreenshotsGalleryState } from "./screenshots-gallery-state"

export function ScreenshotsGalleryErrorState({
  state,
}: {
  state: ScreenshotsGalleryState
}) {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <div className="rounded-full bg-destructive/10 p-3 mb-4">
        <AlertTriangle className="h-10 w-10 text-destructive" />
      </div>
      <h3 className="text-lg font-semibold mb-2">{state.tCommon("status.error")}</h3>
      <p className="text-muted-foreground text-center mb-4">{state.t("loadError")}</p>
      <Button onClick={() => state.refetch()}>{state.tCommon("actions.retry")}</Button>
    </div>
  )
}

export function ScreenshotsGalleryLoadingState() {
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-4">
        <Skeleton className="h-10 w-64" />
      </div>
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
        {Array.from({ length: 8 }).map((_, i) => (
          <Skeleton key={i} className="aspect-video rounded-lg" />
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
      <div className="rounded-full bg-muted p-3 mb-4">
        <ImageIcon className="h-10 w-10 text-muted-foreground" />
      </div>
      <h3 className="text-lg font-semibold mb-2">{state.t("empty.title")}</h3>
      <p className="text-muted-foreground text-center">{state.t("empty.description")}</p>
    </div>
  )
}

export function ScreenshotsGalleryContent({
  state,
}: {
  state: ScreenshotsGalleryState
}) {
  const lightboxScreenshot = state.screenshots[state.lightboxIndex]

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-2">
          <Input
            type="search"
            name="screenshotSearch"
            autoComplete="off"
            placeholder={state.t("filterPlaceholder")}
            value={state.searchInput}
            onChange={(e) => state.setSearchInput(e.target.value)}
            onKeyDown={state.handleKeyDown}
            className="w-64"
          />
          <Button
            variant="outline"
            size="icon"
            onClick={state.handleSearch}
            aria-label={state.tCommon("actions.search")}
          >
            <Search className="h-4 w-4" />
          </Button>
        </div>
        <div className="flex items-center gap-2">
          {state.targetId && state.selectedIds.size > 0 ? (
            <Button variant="destructive" size="sm" onClick={() => state.setDeleteDialogOpen(true)}>
              <Trash2 className="h-4 w-4 mr-1" />
              {state.tCommon("actions.delete")} ({state.selectedIds.size})
            </Button>
          ) : null}
          {state.screenshots.length > 0 && state.targetId ? (
            <Button variant="outline" size="sm" onClick={state.selectAll}>
              {state.selectedIds.size === state.screenshots.length
                ? state.tCommon("actions.deselectAll")
                : state.tCommon("actions.selectAll")}
            </Button>
          ) : null}
        </div>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
        {state.screenshots.map((screenshot, index) => (
          <div
            key={screenshot.id}
            className={cn(
              "group relative aspect-video rounded-lg overflow-hidden border bg-muted transition-[background-color,border-color,box-shadow]",
              state.selectedIds.has(screenshot.id) ? "ring-2 ring-primary" : ""
            )}
          >
            {state.targetId ? (
              <button
                type="button"
                className="absolute top-2 left-2 z-10"
                onClick={(e) => {
                  e.stopPropagation()
                  state.toggleSelect(screenshot.id)
                }}
                aria-label={`Select screenshot ${index + 1}`}
              >
                <Checkbox
                  checked={state.selectedIds.has(screenshot.id)}
                  className="bg-background/80 backdrop-blur-sm"
                />
              </button>
            ) : null}

            <button
              type="button"
              onClick={() => state.openLightbox(index)}
              className="h-full w-full"
              aria-label={`Open screenshot ${index + 1}`}
            >
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={state.getImageUrl(screenshot)}
                alt={screenshot.url}
                className="w-full h-full object-cover transition-transform group-hover:scale-105"
                loading="lazy"
                width={1280}
                height={720}
              />
            </button>

            <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/80 to-transparent p-2">
              <div className="flex items-center gap-2">
                {screenshot.statusCode ? (
                  <span
                    className={cn(
                      "text-xs px-1.5 py-0.5 rounded font-medium shrink-0",
                      screenshot.statusCode >= 200 && screenshot.statusCode < 300 ? "bg-green-500/80 text-white" : "",
                      screenshot.statusCode >= 300 && screenshot.statusCode < 400 ? "bg-blue-500/80 text-white" : "",
                      screenshot.statusCode >= 400 && screenshot.statusCode < 500 ? "bg-yellow-500/80 text-black" : "",
                      screenshot.statusCode >= 500 ? "bg-red-500/80 text-white" : ""
                    )}
                  >
                    {screenshot.statusCode}
                  </span>
                ) : null}
                <p className="text-white text-xs truncate" title={screenshot.url}>
                  {screenshot.url}
                </p>
              </div>
            </div>

            <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity">
              <a
                href={screenshot.url}
                target="_blank"
                rel="noopener noreferrer"
                onClick={(e) => e.stopPropagation()}
                className="inline-flex items-center justify-center h-8 w-8 rounded-md bg-background/80 backdrop-blur-sm hover:bg-background"
                aria-label={state.t("openInNewTab")}
              >
                <ExternalLink className="h-4 w-4" />
              </a>
            </div>
          </div>
        ))}
      </div>

      {state.screenshots.length === 0 && state.filterQuery ? (
        <div className="flex flex-col items-center justify-center py-12">
          <p className="text-muted-foreground">{state.t("noResults")}</p>
        </div>
      ) : null}

      {(state.paginationInfo.totalPages > 1 || state.paginationInfo.total > state.pagination.pageSize) ? (
        <div className="flex justify-center items-center gap-4">
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                state.setPagination((prev) => ({ ...prev, pageIndex: Math.max(0, prev.pageIndex - 1) }))
              }
              disabled={state.pagination.pageIndex === 0}
              aria-label={state.t("previousPage")}
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <span className="text-sm text-muted-foreground">
              {state.pagination.pageIndex + 1} / {state.displayTotalPages}
            </span>
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                state.setPagination((prev) => ({
                  ...prev,
                  pageIndex: Math.min(state.maxPageIndex, prev.pageIndex + 1),
                }))
              }
              disabled={state.pagination.pageIndex >= state.maxPageIndex}
              aria-label={state.t("nextPage")}
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
          <Select value={String(state.pagination.pageSize)} onValueChange={state.handlePageSizeChange}>
            <SelectTrigger className="w-20 h-8">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {state.pageSizeOptions.map((size) => (
                <SelectItem key={size} value={String(size)}>
                  {size}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      ) : null}

      <Dialog open={state.lightboxOpen} onOpenChange={state.setLightboxOpen}>
        <DialogContent className="max-w-[90vw] max-h-[90vh] p-0 bg-black/95 border-none">
          <VisuallyHidden>
            <DialogTitle>{state.t("lightboxTitle")}</DialogTitle>
          </VisuallyHidden>
          <div className="relative w-full h-full flex items-center justify-center">
            <button type="button"
              onClick={() => state.setLightboxOpen(false)}
              className="absolute top-4 right-4 z-50 p-2 rounded-full bg-white/10 hover:bg-white/20 transition-colors"
              aria-label={state.t("closeLightbox")}
            >
              <X className="h-6 w-6 text-white" />
            </button>

            {state.screenshots.length > 1 ? (
              <>
                <button type="button"
                  onClick={state.prevImage}
                  className="absolute left-4 z-50 p-2 rounded-full bg-white/10 hover:bg-white/20 transition-colors"
                  aria-label={state.t("previousScreenshot")}
                >
                  <ChevronLeft className="h-8 w-8 text-white" />
                </button>
                <button type="button"
                  onClick={state.nextImage}
                  className="absolute right-4 z-50 p-2 rounded-full bg-white/10 hover:bg-white/20 transition-colors"
                  aria-label={state.t("nextScreenshot")}
                >
                  <ChevronRight className="h-8 w-8 text-white" />
                </button>
              </>
            ) : null}

            {lightboxScreenshot ? (
              <div className="flex flex-col items-center gap-4 p-8">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src={state.getImageUrl(lightboxScreenshot)}
                  alt={lightboxScreenshot.url}
                  className="max-w-full max-h-[70vh] object-contain"
                  width={1920}
                  height={1080}
                />
                <div className="text-white text-center">
                  <p className="text-sm opacity-80">
                    {state.lightboxIndex + 1} / {state.screenshots.length}
                  </p>
                  <div className="flex items-center gap-2 justify-center mt-1">
                    {lightboxScreenshot.statusCode ? (
                      <span
                        className={cn(
                          "text-sm px-2 py-0.5 rounded font-medium",
                          lightboxScreenshot.statusCode >= 200 &&
                            lightboxScreenshot.statusCode < 300
                            ? "bg-green-500 text-white"
                            : "",
                          lightboxScreenshot.statusCode >= 300 &&
                            lightboxScreenshot.statusCode < 400
                            ? "bg-blue-500 text-white"
                            : "",
                          lightboxScreenshot.statusCode >= 400 &&
                            lightboxScreenshot.statusCode < 500
                            ? "bg-yellow-500 text-black"
                            : "",
                          lightboxScreenshot.statusCode >= 500
                            ? "bg-red-500 text-white"
                            : ""
                        )}
                      >
                        {lightboxScreenshot.statusCode}
                      </span>
                    ) : null}
                    <a
                      href={lightboxScreenshot.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-400 hover:underline flex items-center gap-1"
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
