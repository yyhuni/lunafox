export const SCAN_OVERVIEW_WORKBENCH_CLASS =
  "flex min-h-0 min-w-0 flex-1 flex-col gap-4 xl:flex-row xl:items-start"

export const SCAN_OVERVIEW_PRIMARY_COLUMN_CLASS = "flex min-w-0 flex-1 flex-col gap-4"

export const SCAN_OVERVIEW_SIDE_PANEL_CLASS = "w-full shrink-0 xl:w-80"

export const SCAN_OVERVIEW_STICKY_SIDE_PANEL_CLASS = `${SCAN_OVERVIEW_SIDE_PANEL_CLASS} xl:sticky xl:top-4`

export const SCAN_OVERVIEW_SUMMARY_GRID_CLASS = "grid min-w-0 grid-cols-2 sm:grid-cols-6"

export const SCAN_OVERVIEW_SUMMARY_ITEM_CLASS =
  "flex min-w-0 items-baseline justify-between gap-1.5 px-3 py-2 sm:justify-center"

export function getScanOverviewSummaryItemDividerClass(index: number) {
  return [
    index > 1 ? "border-t border-border/60 sm:border-t-0" : null,
    index % 2 === 1 ? "border-l border-border/60" : null,
    index > 0 ? "sm:border-l sm:border-border/60" : null,
  ]
}
