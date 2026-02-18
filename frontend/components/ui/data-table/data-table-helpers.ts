import type { DownloadOption } from "@/types/data-table.types"

type DownloadCallbacks = {
  onDownloadAll?: () => void
  onDownloadSelected?: () => void
  onDownloadImportant?: () => void
  onDownloadInteresting?: () => void
}

type DownloadTranslator = (key: string, params?: Record<string, string | number | Date>) => string

export function buildDownloadOptions(
  tDownload: DownloadTranslator,
  {
    onDownloadAll,
    onDownloadSelected,
    onDownloadImportant,
    onDownloadInteresting,
  }: DownloadCallbacks
): DownloadOption[] {
  const options: DownloadOption[] = []

  if (onDownloadAll) {
    options.push({
      key: "all",
      label: tDownload("all"),
      onClick: onDownloadAll,
    })
  }

  if (onDownloadSelected) {
    options.push({
      key: "selected",
      label: tDownload("selected"),
      onClick: onDownloadSelected,
      disabled: (count) => count === 0,
    })
  }

  if (onDownloadImportant) {
    options.push({
      key: "important",
      label: tDownload("important"),
      onClick: onDownloadImportant,
    })
  }

  if (onDownloadInteresting) {
    options.push({
      key: "interesting",
      label: tDownload("interesting"),
      onClick: onDownloadInteresting,
    })
  }

  return options
}
