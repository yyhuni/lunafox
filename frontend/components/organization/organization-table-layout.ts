import {
  getFixedTableLayoutTotalWidth,
  type FixedTableColumnLayout,
} from "@/lib/ui/table-layout-contract"

export const organizationTableColumnLayout = {
  select: { key: "select", size: 40, minSize: 40, maxSize: 40 },
  name: { key: "name", size: 200, minSize: 150 },
  targetCount: { key: "targetCount", size: 120, minSize: 80, maxSize: 140 },
  createdAt: { key: "createdAt", size: 176, minSize: 176, maxSize: 220 },
  actions: { key: "actions", size: 88, minSize: 88, maxSize: 88 },
} as const satisfies Record<string, FixedTableColumnLayout>

export const ORGANIZATION_TABLE_COLUMN_LAYOUT = Object.values(organizationTableColumnLayout)
export const ORGANIZATION_TABLE_MIN_WIDTH_PX = getFixedTableLayoutTotalWidth(ORGANIZATION_TABLE_COLUMN_LAYOUT)
