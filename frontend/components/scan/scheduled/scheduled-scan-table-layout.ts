import {
  getFixedTableLayoutTotalWidth,
  type FixedTableColumnLayout,
} from "@/lib/ui/table-layout-contract"

export const scheduledScanTableColumnLayout = {
  select: { key: "select", size: 40, minSize: 40, maxSize: 40 },
  name: { key: "name", size: 250, minSize: 200 },
  cronExpression: { key: "cronExpression", size: 150, minSize: 130 },
  scanMode: { key: "scanMode", size: 200, minSize: 150 },
  isEnabled: { key: "isEnabled", size: 120, minSize: 100 },
  nextRunTime: { key: "nextRunTime", size: 176, minSize: 176, maxSize: 220 },
  handoffResults: { key: "handoffResults", size: 256, minSize: 232, maxSize: 300 },
  lastRunTime: { key: "lastRunTime", size: 176, minSize: 176, maxSize: 220 },
  actions: { key: "actions", size: 120, minSize: 120, maxSize: 120 },
} as const satisfies Record<string, FixedTableColumnLayout>

export const SCHEDULED_SCAN_TABLE_COLUMN_LAYOUT = Object.values(scheduledScanTableColumnLayout)
export const SCHEDULED_SCAN_TABLE_MIN_WIDTH_PX = getFixedTableLayoutTotalWidth(
  SCHEDULED_SCAN_TABLE_COLUMN_LAYOUT
)
