import {
  getFixedTableLayoutTotalWidth,
  type FixedTableColumnLayout,
} from "@/lib/ui/table-layout-contract"

export const vulnerabilitiesTableColumnLayout = {
  select: { key: "select", size: 40, minSize: 40, maxSize: 40 },
  reviewStatus: { key: "reviewStatus", size: 56, minSize: 56, maxSize: 56 },
  severity: { key: "severity", size: 100, minSize: 80, maxSize: 120 },
  source: { key: "source", size: 100, minSize: 80, maxSize: 150 },
  vulnType: { key: "vulnType", size: 150, minSize: 100, maxSize: 250 },
  url: { key: "url", size: 500, minSize: 300, maxSize: 700 },
  createdAt: { key: "createdAt", size: 176, minSize: 176, maxSize: 220 },
} as const satisfies Record<string, FixedTableColumnLayout>

export const VULNERABILITIES_TABLE_COLUMN_LAYOUT = Object.values(vulnerabilitiesTableColumnLayout)
export const VULNERABILITIES_TABLE_MIN_WIDTH_PX = getFixedTableLayoutTotalWidth(
  VULNERABILITIES_TABLE_COLUMN_LAYOUT
)
