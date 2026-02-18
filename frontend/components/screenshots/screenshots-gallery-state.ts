import React from "react"
import { useTranslations } from "next-intl"
import { toast } from "sonner"

import { useTargetScreenshots, useScanScreenshots } from "@/hooks/use-screenshots"
import { buildPaginationInfo } from "@/hooks/_shared/pagination"
import { ScreenshotService } from "@/services/screenshot.service"

const PAGE_SIZE_OPTIONS = [12, 24, 48]

interface Screenshot {
  id: number
  url: string
  statusCode: number | null
  createdAt: string
}

interface ScreenshotsGalleryStateOptions {
  targetId?: number
  scanId?: number
}

export function useScreenshotsGalleryState({
  targetId,
  scanId,
}: ScreenshotsGalleryStateOptions) {
  const [pagination, setPagination] = React.useState({ pageIndex: 0, pageSize: 12 })
  const [searchInput, setSearchInput] = React.useState("")
  const [filterQuery, setFilterQuery] = React.useState("")
  const [selectedIds, setSelectedIds] = React.useState<Set<number>>(new Set())
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [isDeleting, setIsDeleting] = React.useState(false)
  const [lightboxOpen, setLightboxOpen] = React.useState(false)
  const [lightboxIndex, setLightboxIndex] = React.useState(0)

  const t = useTranslations("pages.screenshots")
  const tCommon = useTranslations("common")
  const tToast = useTranslations("toast")

  const targetQuery = useTargetScreenshots(
    targetId || 0,
    {
      page: pagination.pageIndex + 1,
      pageSize: pagination.pageSize,
      filter: filterQuery || undefined,
    },
    { enabled: !!targetId }
  )

  const scanQuery = useScanScreenshots(
    scanId || 0,
    {
      page: pagination.pageIndex + 1,
      pageSize: pagination.pageSize,
      filter: filterQuery || undefined,
    },
    { enabled: !!scanId }
  )

  const activeQuery = targetId ? targetQuery : scanQuery
  const { data, isLoading, error, refetch } = activeQuery

  const screenshots: Screenshot[] = React.useMemo(() => data?.results || [], [data])
  const total = data?.total ?? screenshots.length ?? 0

  const paginationInfo = buildPaginationInfo({
    total,
    page: pagination.pageIndex + 1,
    pageSize: pagination.pageSize,
    totalPages: data?.totalPages || undefined,
    minTotalPages: total > 0 ? 1 : 0,
  })

  const maxPageIndex = Math.max(0, paginationInfo.totalPages - 1)
  const displayTotalPages = Math.max(1, paginationInfo.totalPages)

  const toggleSelect = React.useCallback((id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }, [])

  const selectAll = React.useCallback(() => {
    if (selectedIds.size === screenshots.length) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(screenshots.map((s) => s.id)))
    }
  }, [screenshots, selectedIds.size])

  const handleBulkDelete = async () => {
    if (selectedIds.size === 0) return
    setIsDeleting(true)
    try {
      const result = await ScreenshotService.bulkDelete(Array.from(selectedIds))
      toast.success(tToast("deleteSuccess", { count: result.deletedCount }))
      setSelectedIds(new Set())
      setDeleteDialogOpen(false)
      refetch()
    } catch {
      toast.error(tToast("deleteFailed"))
    } finally {
      setIsDeleting(false)
    }
  }

  const handleSearch = () => {
    setFilterQuery(searchInput)
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      handleSearch()
    }
  }

  const handlePageSizeChange = (value: string) => {
    const newPageSize = parseInt(value, 10)
    setPagination({ pageIndex: 0, pageSize: newPageSize })
  }

  const openLightbox = (index: number) => {
    setLightboxIndex(index)
    setLightboxOpen(true)
  }

  const nextImage = () => {
    setLightboxIndex((prev) => (prev + 1) % screenshots.length)
  }

  const prevImage = () => {
    setLightboxIndex((prev) => (prev - 1 + screenshots.length) % screenshots.length)
  }

  const getImageUrl = React.useCallback(
    (screenshot: Screenshot) => {
      if (scanId) {
        return ScreenshotService.getSnapshotImageUrl(scanId, screenshot.id)
      }
      return ScreenshotService.getImageUrl(screenshot.id)
    },
    [scanId]
  )

  return {
    t,
    tCommon,
    targetId,
    isLoading,
    error,
    refetch,
    data,
    screenshots,
    filterQuery,
    searchInput,
    setSearchInput,
    pagination,
    setPagination,
    paginationInfo,
    maxPageIndex,
    displayTotalPages,
    selectedIds,
    deleteDialogOpen,
    setDeleteDialogOpen,
    isDeleting,
    lightboxOpen,
    setLightboxOpen,
    lightboxIndex,
    pageSizeOptions: PAGE_SIZE_OPTIONS,
    toggleSelect,
    selectAll,
    handleBulkDelete,
    handleSearch,
    handleKeyDown,
    handlePageSizeChange,
    openLightbox,
    nextImage,
    prevImage,
    getImageUrl,
  }
}

export type ScreenshotsGalleryState = ReturnType<typeof useScreenshotsGalleryState>
