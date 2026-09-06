import {
  getFixedTableLayoutTotalWidth,
  type FixedTableColumnLayout,
} from "@/lib/ui/table-layout-contract"

export const allTargetsTableColumnLayout = {
  select: { key: "select", size: 40, minSize: 40, maxSize: 40 },
  name: { key: "name", size: 300, minSize: 220 },
  organizations: { key: "organizations", size: 200, minSize: 150, maxSize: 350 },
  createdAt: { key: "createdAt", size: 176, minSize: 176, maxSize: 220 },
  lastScannedAt: { key: "lastScannedAt", size: 176, minSize: 176, maxSize: 220 },
  actions: { key: "actions", size: 88, minSize: 88, maxSize: 88 },
} as const satisfies Record<string, FixedTableColumnLayout>

export const ALL_TARGETS_TABLE_COLUMN_LAYOUT = Object.values(allTargetsTableColumnLayout)
export const ALL_TARGETS_TABLE_MIN_WIDTH_PX = getFixedTableLayoutTotalWidth(ALL_TARGETS_TABLE_COLUMN_LAYOUT)
