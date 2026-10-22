// Asset type counts are heavy-tailed (URLs are page-level while sites are
// domain-level), so the overview never compares them by bar length. Each type
// gets its own sparkline, which carries only that type's seven-day shape.
//
// The viewBox stays fixed and the rendered width is fluid, so the geometry is
// independent of the panel width and the same points work at every breakpoint.
export const ASSET_SPARKLINE_VIEW_WIDTH = 100
export const ASSET_SPARKLINE_VIEW_HEIGHT = 24

const VERTICAL_PADDING = 2
const ROUNDING_FACTOR = 100

function roundCoordinate(value: number): number {
  return Math.round(value * ROUNDING_FACTOR) / ROUNDING_FACTOR
}

function midpointY(height: number): number {
  return VERTICAL_PADDING + (height - VERTICAL_PADDING * 2) / 2
}

/**
 * Builds the filled area under a sparkline so the plot reads as a soft band
 * instead of a bare line. The area closes on the viewBox floor.
 */
export function buildAssetSparklineAreaPoints(
  points: string,
  width = ASSET_SPARKLINE_VIEW_WIDTH,
  height = ASSET_SPARKLINE_VIEW_HEIGHT,
): string {
  if (!points) {
    return ""
  }

  return `0,${height} ${points} ${width},${height}`
}

/**
 * Builds the polyline points for one asset type's seven-day history.
 *
 * A single point or a perfectly flat series renders as a horizontal line at
 * the vertical center: there is no trend shape to draw, and inventing a slope
 * would misrepresent the data.
 */
export function buildAssetSparklinePoints(
  values: readonly number[],
  width = ASSET_SPARKLINE_VIEW_WIDTH,
  height = ASSET_SPARKLINE_VIEW_HEIGHT,
): string {
  if (values.length === 0) {
    return ""
  }

  const centerY = midpointY(height)

  if (values.length === 1) {
    return `0,${roundCoordinate(centerY)} ${width},${roundCoordinate(centerY)}`
  }

  const min = Math.min(...values)
  const max = Math.max(...values)
  const range = max - min
  const usableHeight = height - VERTICAL_PADDING * 2
  const stepX = width / (values.length - 1)

  return values
    .map((value, index) => {
      const x = roundCoordinate(index * stepX)
      const y = range === 0
        ? roundCoordinate(centerY)
        : roundCoordinate(VERTICAL_PADDING + ((max - value) / range) * usableHeight)

      return `${x},${y}`
    })
    .join(" ")
}
