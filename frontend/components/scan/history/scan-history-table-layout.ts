import {
  getFixedTableLayoutTotalWidth,
  type FixedTableColumnLayout,
} from "@/lib/ui/table-layout-contract"

export const scanHistoryTableColumnLayout = {
  select: { key: "select", size: 40, minSize: 40, maxSize: 40 },
  // The target cell leads with the compact trigger provenance marker.
  target: { key: "target", size: 220, minSize: 160, maxSize: 280 },
  cachedStats: { key: "cachedStats", size: 240, minSize: 160, maxSize: 320 },
  executedEngines: { key: "executedEngines", size: 220, minSize: 152, maxSize: 280 },
  createdAt: { key: "createdAt", size: 176, minSize: 176, maxSize: 200 },
  status: { key: "status", size: 112, minSize: 108, maxSize: 140 },
  actions: { key: "actions", size: 56, minSize: 56, maxSize: 56 },
} as const satisfies Record<string, FixedTableColumnLayout>

export const SCAN_HISTORY_TABLE_COLUMN_LAYOUT = Object.values(scanHistoryTableColumnLayout)
export const SCAN_HISTORY_TABLE_MIN_WIDTH_PX = getFixedTableLayoutTotalWidth(
  SCAN_HISTORY_TABLE_COLUMN_LAYOUT
)
export const SCAN_HISTORY_WITHOUT_TARGET_TABLE_COLUMN_LAYOUT =
  SCAN_HISTORY_TABLE_COLUMN_LAYOUT.filter((column) => column.key !== "target")
export const SCAN_HISTORY_WITHOUT_TARGET_TABLE_MIN_WIDTH_PX = getFixedTableLayoutTotalWidth(
  SCAN_HISTORY_WITHOUT_TARGET_TABLE_COLUMN_LAYOUT
)
