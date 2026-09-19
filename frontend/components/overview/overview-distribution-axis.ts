import { getUiSansCanvasFont } from "@/lib/font-stacks"
import { measureTextWidth } from "@/lib/table-utils"

// The category labels come from i18n, so the widest one is locale-owned: English needs
// ~70px for "Subdomains" while Chinese needs 36px for "子域名". One fixed column sized
// for the longer locale leaves the shorter labels floating behind an empty band, so the
// column is sized from the labels that are actually rendered.
//
// Measuring the real font is the only way to stay correct for both the locale and the
// platform font stack. It is safe here because the resolved overview sections load with
// `ssr: false`, so no server-rendered width can disagree with the measured one.
const AXIS_TICK_FONT_SIZE_PX = 12
const AXIS_TICK_FONT = getUiSansCanvasFont({ sizePx: AXIS_TICK_FONT_SIZE_PX })

export const ASSET_DISTRIBUTION_AXIS_TICK_MARGIN = 8
// Recharts keeps its default tick offset even with the tick line disabled, so label text
// ends this far before the bar area's left edge.
export const ASSET_DISTRIBUTION_AXIS_TICK_OFFSET = 6

// Keep the widest label clear of the panel edge, and bound the column so a single long
// localized label cannot take over a narrow chart column.
const AXIS_LABEL_LEADING_GAP_PX = 4
const AXIS_MIN_WIDTH_PX = 56
const AXIS_MAX_WIDTH_PX = 96

export function resolveAssetDistributionAxisWidth(labels: readonly string[]): number {
  const widestLabel = labels.reduce(
    (widest, label) => Math.max(widest, measureTextWidth(label, AXIS_TICK_FONT)),
    0,
  )
  const requiredWidth =
    widestLabel +
    ASSET_DISTRIBUTION_AXIS_TICK_MARGIN +
    ASSET_DISTRIBUTION_AXIS_TICK_OFFSET +
    AXIS_LABEL_LEADING_GAP_PX

  return Math.min(AXIS_MAX_WIDTH_PX, Math.max(AXIS_MIN_WIDTH_PX, Math.ceil(requiredWidth)))
}
