import type { ExportOption } from "@/types/data-table.types"

type ExportCallbacks = {
  onExportAll?: () => void
  onExportSelected?: () => void
  onExportImportant?: () => void
  onExportInteresting?: () => void
}

type ExportTranslator = (key: string, params?: Record<string, string | number | Date>) => string

export function buildExportOptions(
  tExport: ExportTranslator,
  {
    onExportAll,
    onExportSelected,
    onExportImportant,
    onExportInteresting,
  }: ExportCallbacks
): ExportOption[] {
  const options: ExportOption[] = []

  if (onExportAll) {
    options.push({
      key: "all",
      label: tExport("all"),
      onClick: onExportAll,
    })
  }

  if (onExportSelected) {
    options.push({
      key: "selected",
      label: tExport("selected"),
      onClick: onExportSelected,
      disabled: (count) => count === 0,
    })
  }

  if (onExportImportant) {
    options.push({
      key: "important",
      label: tExport("important"),
      onClick: onExportImportant,
    })
  }

  if (onExportInteresting) {
    options.push({
      key: "interesting",
      label: tExport("interesting"),
      onClick: onExportInteresting,
    })
  }

  return options
}
